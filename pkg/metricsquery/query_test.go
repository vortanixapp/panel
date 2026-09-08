package metricsquery

import "testing"

// Ведро должно быть таким, чтобы окно целиком уложилось в лимит строк. Раньше
// свёртки не было вовсе, точки шли сырыми, и лимит в 500 отрезал начало окна:
// история за неделю показывала первые два часа.
func TestBucketFitsWindowIntoLimit(t *testing.T) {
	const limit = 500
	for _, hours := range []int{1, 6, 24, 24 * 7, 24 * 30, 24 * 90, 24 * 180} {
		bucket := BucketSeconds(hours)
		if bucket <= 0 {
			t.Fatalf("окно %d ч: ведро не задано", hours)
		}
		rows := hours * 3600 / bucket
		if rows > limit {
			t.Errorf("окно %d ч: %d строк при ведре %d с — не помещается в лимит %d",
				hours, rows, bucket, limit)
		}
	}
}

// Соседние окна не должны схлопываться в одно ведро: чем длиннее период, тем
// крупнее шаг, но порядок обязан сохраняться.
func TestBucketGrowsWithWindow(t *testing.T) {
	prev := 0
	for _, hours := range []int{1, 6, 24, 24 * 7, 24 * 30, 24 * 90, 24 * 365} {
		bucket := BucketSeconds(hours)
		if bucket < prev {
			t.Errorf("окно %d ч: ведро %d с меньше предыдущего %d с", hours, bucket, prev)
		}
		prev = bucket
	}
}

// Часовая история строится по вёдрам не крупнее часа, иначе строки таблицы
// склеятся: панель раскладывает ответ по своим вёдрам поверх наших.
func TestWeekBucketNotCoarserThanHour(t *testing.T) {
	if got := BucketSeconds(24 * 7); got > 3600 {
		t.Errorf("для недели ведро %d с крупнее часа — строки истории склеятся", got)
	}
}
