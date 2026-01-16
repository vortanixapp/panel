import './bootstrap';
import Chart from 'chart.js/auto';

window.Chart = Chart;

(() => {
    const nodes = Array.from(document.querySelectorAll('[data-reveal]'));
    if (nodes.length === 0) return;

    for (const el of nodes) {
        el.classList.add('vtx-reveal');
    }

    const io = new IntersectionObserver(
        (entries) => {
            for (const e of entries) {
                if (!e.isIntersecting) continue;

                const el = e.target;
                const delay = Number(el.getAttribute('data-delay') || '0') || 0;
                el.style.transitionDelay = `${Math.max(0, delay)}ms`;
                el.classList.add('is-visible');
                io.unobserve(el);
            }
        },
        {
            root: null,
            rootMargin: '0px 0px -10% 0px',
            threshold: 0.08,
        }
    );

    for (const el of nodes) io.observe(el);
})();

(() => {
    const root = document.querySelector('[data-hero-carousel]');
    if (!root) return;

    const slides = Array.from(root.querySelectorAll('[data-hero-slide]'));
    if (slides.length <= 1) return;

    const prefersReduced = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const intervalMs = Number(root.getAttribute('data-interval') || '6500') || 6500;

    let idx = 0;
    let timer = null;

    function apply(i) {
        for (let k = 0; k < slides.length; k++) {
            slides[k].classList.toggle('is-active', k === i);
        }
    }

    function next() {
        idx = (idx + 1) % slides.length;
        apply(idx);
    }

    function start() {
        if (prefersReduced) {
            apply(0);
            return;
        }
        if (timer) return;
        timer = window.setInterval(next, Math.max(2000, intervalMs));
    }

    function stop() {
        if (!timer) return;
        window.clearInterval(timer);
        timer = null;
    }

    apply(0);
    start();

    document.addEventListener('visibilitychange', () => {
        if (document.hidden) {
            stop();
        } else {
            start();
        }
    });
})();

(() => {
    const nodes = Array.from(document.querySelectorAll('[data-count]'));
    if (nodes.length === 0) return;

    const prefersReduced = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    function formatWithSuffix(value, suffix) {
        if (!suffix) return String(value);
        return String(value) + suffix;
    }

    function animate(el) {
        const raw = String(el.getAttribute('data-count') || '').trim();
        const suffix = String(el.getAttribute('data-suffix') || '');
        const duration = Math.max(400, Number(el.getAttribute('data-duration') || '900') || 900);

        if (raw === '') return;

        const target = Number(raw);
        if (!Number.isFinite(target)) return;

        if (prefersReduced) {
            el.textContent = formatWithSuffix(target, suffix);
            return;
        }

        const start = performance.now();
        const from = 0;

        function tick(now) {
            const t = Math.min(1, (now - start) / duration);
            const eased = 1 - Math.pow(1 - t, 3);
            const val = Math.round(from + (target - from) * eased);
            el.textContent = formatWithSuffix(val, suffix);
            if (t < 1) requestAnimationFrame(tick);
        }

        requestAnimationFrame(tick);
    }

    const io = new IntersectionObserver(
        (entries) => {
            for (const e of entries) {
                if (!e.isIntersecting) continue;
                animate(e.target);
                io.unobserve(e.target);
            }
        },
        { root: null, rootMargin: '0px 0px -15% 0px', threshold: 0.15 }
    );

    for (const el of nodes) io.observe(el);
})();
