package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type tenantPoolKey struct{}

// withTenantPool кладёт в запрос базу того арендатора, которому он адресован.
// Дальше её достаёт dbOf: так обработчикам и их помощникам не нужно таскать
// пул через сигнатуры, а их около полутысячи.
func (h *Handler) withTenantPool(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := strings.TrimSpace(r.Header.Get("X-Tenant-Slug"))
		if slug == "" || h.tenants == nil {
			next.ServeHTTP(w, r)
			return
		}
		pool, err := h.tenants.Pool(r.Context(), slug)
		if err != nil {
			// Молчать нельзя: запрос уйдёт в центральную базу, где данных
			// арендатора нет, и обработчик отработает «успешно», ничего не
			// изменив.
			log.Printf("база арендатора %q недоступна, отвечает центральная: %v", slug, err)
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tenantPoolKey{}, pool)))
	})
}

// requireTenantPool не пускает дальше запрос, для которого не нашлась база
// арендатора. Раньше такой запрос молча обслуживала центральная база: чтения
// возвращали пустоту, записи падали на внешнем ключе — и всё это выглядело как
// успех. Ни один арендатор в центральной базе больше не живёт, поэтому отказ
// честнее подмены.
func (h *Handler) requireTenantPool(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if _, ok := ctx.Value(tenantPoolKey{}).(*pgxpool.Pool); ok {
			next.ServeHTTP(w, r)
			return
		}
		claims, ok := tenantClaims(ctx)
		if !ok || claims.TenantID == "" || h.tenants == nil {
			next.ServeHTTP(w, r)
			return
		}
		pool := h.tenants.PoolForTenantID(ctx, claims.TenantID)
		if pool == nil {
			log.Printf("база арендатора %s не найдена — запрос отклонён", claims.TenantID)
			writeError(w, http.StatusServiceUnavailable, "база арендатора недоступна")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, tenantPoolKey{}, pool)))
	})
}

// withTenantSlug выбирает базу по слагу, известному только самому обработчику:
// у входа он приходит в теле запроса, а не заголовком.
func (h *Handler) withTenantSlug(ctx context.Context, slug string) context.Context {
	if h.tenants == nil || strings.TrimSpace(slug) == "" {
		return ctx
	}
	pool, err := h.tenants.Pool(ctx, slug)
	if err != nil {
		return ctx
	}
	return context.WithValue(ctx, tenantPoolKey{}, pool)
}

// dbOf отдаёт базу арендатора из запроса. Пока арендатор не переехал в свою
// базу, отвечает центральная — это позволяет переводить обработчики по одному.
func (h *Handler) dbOf(ctx context.Context) *pgxpool.Pool {
	if pool, ok := ctx.Value(tenantPoolKey{}).(*pgxpool.Pool); ok && pool != nil {
		return pool
	}
	if pool := h.poolByClaims(ctx); pool != nil {
		return pool
	}
	return h.db
}

// poolByClaims выбирает базу по арендатору из токена. Middleware ставит пул по
// заголовку и работает до авторизации, поэтому без заголовка база оставалась
// центральной — и запись уходила туда, где этого арендатора нет.
func (h *Handler) poolByClaims(ctx context.Context) *pgxpool.Pool {
	if h.tenants == nil {
		return nil
	}
	claims, ok := tenantClaims(ctx)
	if !ok || claims.TenantID == "" {
		return nil
	}
	return h.tenants.PoolForTenantID(ctx, claims.TenantID)
}

// readerOf — то же, что dbOf, но для чтения: у арендатора одна база, поэтому
// реплика центральной ему не подходит.
func (h *Handler) readerOf(ctx context.Context) *pgxpool.Pool {
	if pool, ok := ctx.Value(tenantPoolKey{}).(*pgxpool.Pool); ok && pool != nil {
		return pool
	}
	if pool := h.poolByClaims(ctx); pool != nil {
		return pool
	}
	return h.reader()
}
