package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// агрегат по автору
type agg struct {
	AuthorName  string
	AuthorEmail string
	Commits     int
	Added       int
	Deleted     int
}

func (a agg) Net() int { return a.Added - a.Deleted }

func main() {
	var (
		repoPath      string
		sinceStr      string
		untilStr      string
		authorFilter  string
		includeMerges bool
		csvPath       string
	)
	flag.StringVar(&repoPath, "repo", ".", "путь к git-репозиторию")
	flag.StringVar(&sinceStr, "since", "", "начало периода (YYYY-MM-DD или RFC3339, локально Europe/Moscow)")
	flag.StringVar(&untilStr, "until", "", "конец периода (YYYY-MM-DD или RFC3339, локально Europe/Moscow)")
	flag.StringVar(&authorFilter, "author", "", "фильтр по автору (substring: имя/емейл, case-insensitive)")
	flag.BoolVar(&includeMerges, "include-merges", false, "включать merge-коммиты (по умолчанию false)")
	flag.StringVar(&csvPath, "csv", "", "сохранить итог в CSV (путь к файлу)")
	flag.Parse()

	if repoPath == "" {
		log.Fatal("укажи --repo")
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	since, until, err := parsePeriod(sinceStr, untilStr, loc)
	if err != nil {
		log.Fatalf("ошибка парсинга дат: %v", err)
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		log.Fatalf("не удалось открыть репо: %v", err)
	}

	logOpts := &git.LogOptions{
		Since: &since,
		Until: &until,
		// CommitterTime — ближе к привычному «по времени коммиттера»
		Order: git.LogOrderCommitterTime,
	}

	iter, err := repo.Log(logOpts)
	if err != nil {
		log.Fatalf("git log error: %v", err)
	}
	defer iter.Close()

	byAuthor := map[string]*agg{}
	totalCommits := 0
	totalAdded := 0
	totalDeleted := 0

	err = iter.ForEach(func(c *object.Commit) error {
		// фильтр по автору
		if authorFilter != "" {
			needle := strings.ToLower(authorFilter)
			if !strings.Contains(strings.ToLower(c.Author.Name), needle) &&
				!strings.Contains(strings.ToLower(c.Author.Email), needle) {
				return nil
			}
		}
		// пропустить merge-коммиты, если не включили
		if !includeMerges && c.NumParents() > 1 {
			return nil
		}

		// Статистика изменений относительно родителя (или пустого дерева для корня)
		stats, err := c.Stats()
		if err != nil {
			// бывают редкие артефакты/битые коммиты — не уроняем всё
			log.Printf("warn: stats for %s: %v", c.Hash.String(), err)
			return nil
		}

		added, deleted := 0, 0
		for _, s := range stats {
			added += s.Addition
			deleted += s.Deletion
		}

		key := strings.ToLower(strings.TrimSpace(c.Author.Email))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(c.Author.Name))
		}
		if key == "" {
			key = "(unknown)"
		}

		rec, ok := byAuthor[key]
		if !ok {
			rec = &agg{AuthorName: c.Author.Name, AuthorEmail: c.Author.Email}
			byAuthor[key] = rec
		}
		rec.Commits++
		rec.Added += added
		rec.Deleted += deleted

		totalCommits++
		totalAdded += added
		totalDeleted += deleted
		return nil
	})
	if err != nil {
		log.Fatalf("обход логов прерван: %v", err)
	}

	// сортировка: по net (desc), затем по коммитам
	list := make([]*agg, 0, len(byAuthor))
	for _, v := range byAuthor {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Net() != list[j].Net() {
			return list[i].Net() > list[j].Net()
		}
		if list[i].Commits != list[j].Commits {
			return list[i].Commits > list[j].Commits
		}
		return strings.ToLower(list[i].AuthorEmail) < strings.ToLower(list[j].AuthorEmail)
	})

	// вывод
	fmt.Printf("Repo: %s\n", repoPath)
	fmt.Printf("Period: %s .. %s (%s)\n",
		since.In(loc).Format(time.RFC3339),
		until.In(loc).Format(time.RFC3339),
		loc)
	if authorFilter != "" {
		fmt.Printf("Author filter: %q\n", authorFilter)
	}
	if !includeMerges {
		fmt.Println("Merges: excluded")
	} else {
		fmt.Println("Merges: included")
	}
	fmt.Println()

	fmt.Printf("%-28s  %-6s  %-8s  %-8s  %-8s\n", "Author", "Commits", "Added", "Deleted", "Net")
	fmt.Println(strings.Repeat("-", 28+2+6+2+8+2+8+2+8))
	for _, a := range list {
		name := a.AuthorName
		if name == "" {
			name = a.AuthorEmail
		}
		if name == "" {
			name = "(unknown)"
		}
		if len(name) > 28 {
			name = name[:25] + "..."
		}
		fmt.Printf("%-28s  %6d  %8d  %8d  %8d\n", name, a.Commits, a.Added, a.Deleted, a.Net())
	}
	fmt.Println(strings.Repeat("-", 28+2+6+2+8+2+8+2+8))
	fmt.Printf("%-28s  %6d  %8d  %8d  %8d\n", "TOTAL", totalCommits, totalAdded, totalDeleted, totalAdded-totalDeleted)

	// CSV (если попросили)
	if csvPath != "" {
		if err := writeCSV(csvPath, list, totalCommits, totalAdded, totalDeleted); err != nil {
			log.Fatalf("csv: %v", err)
		}
		fmt.Printf("\nCSV сохранён: %s\n", csvPath)
	}
}

func parsePeriod(sinceStr, untilStr string, loc *time.Location) (time.Time, time.Time, error) {
	now := time.Now().In(loc)
	var since time.Time
	var until time.Time
	var err error

	if sinceStr == "" {
		// дефолт: 30 дней назад с 00:00
		s := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -30)
		since = s
	} else {
		since, err = parseTimeFlexible(sinceStr, loc, true)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("since: %w", err)
		}
	}

	if untilStr == "" {
		// дефолт: текущий момент
		until = now
	} else {
		until, err = parseTimeFlexible(untilStr, loc, false)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("until: %w", err)
		}
	}
	return since, until, nil
}

func parseTimeFlexible(s string, loc *time.Location, startOfDay bool) (time.Time, error) {
	s = strings.TrimSpace(s)
	// Пытаемся RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// YYYY-MM-DD
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		if startOfDay {
			return t, nil
		}
		// конец дня 23:59:59
		return t.Add(23*time.Hour + 59*time.Minute + 59*time.Second), nil
	}
	// YYYY-MM-DD HH:MM
	if t, err := time.ParseInLocation("2006-01-02 15:04", s, loc); err == nil {
		return t, nil
	}
	// fallback — как есть (может быть с timezone аббревиатурой)
	if t, err := time.ParseInLocation(time.RFC1123Z, s, loc); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("не удалось распарсить %q", s)
}

func writeCSV(path string, rows []*agg, totalCommits, totalAdded, totalDeleted int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	_ = w.Write([]string{"author_name", "author_email", "commits", "added", "deleted", "net"})
	for _, a := range rows {
		_ = w.Write([]string{
			a.AuthorName,
			a.AuthorEmail,
			fmt.Sprintf("%d", a.Commits),
			fmt.Sprintf("%d", a.Added),
			fmt.Sprintf("%d", a.Deleted),
			fmt.Sprintf("%d", a.Net()),
		})
	}
	// Итого как последняя строка
	_ = w.Write([]string{
		"TOTAL", "",
		fmt.Sprintf("%d", totalCommits),
		fmt.Sprintf("%d", totalAdded),
		fmt.Sprintf("%d", totalDeleted),
		fmt.Sprintf("%d", totalAdded-totalDeleted),
	})
	return w.Error()
}
