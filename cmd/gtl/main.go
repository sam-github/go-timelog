package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const timelog = ".gtimelog/timelog.txt"

func main() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err.Error())
	}

	timelog := filepath.Join(usr.HomeDir, timelog)

	f, err := os.Open(timelog)

	if err != nil {
		log.Fatal(err.Error())
	}

	/*
		Here is a formal grammar:

		file ::= (entry|day-separator|comment|old-style-comment)*

		entry ::= timestamp ":" SPACE title NEWLINE

		day-separator ::= NEWLINE

		comment ::= "#" anything* NEWLINE

		old-style-comment ::= anything* NEWLINE

		title ::= anything*
		timestamp is YYYY-MM-DD HH:MM with a single space between the date and the time.

		anything is any character except a newline.

		NEWLINE is whatever Python considers it to be (i.e. CR LF or just LF).

		GTimeLog adds a blank line between days. It ignores them when loading, but this is likely to change in the future.

		GTimeLog considers any lines not starting with a valid timestamp to be comments. This is likely to change in the future, so please use '#' to indicate real comments if you find you need them.

		All lines should be sorted by time. Currently GTimeLog won't complain if they're not, and it will sort them to compensate.
	*/
	// YYYY-MM-DD HH:MM: TITLE
	rx := regexp.MustCompile(`(\d\d\d\d-\d\d-\d\d \d\d:\d\d): (.*)`)

	// Time format
	tf := "2006-01-02 15:04"

	var current WeekReport

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		match := rx.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}

		title := match[2]
		dt, err := time.Parse(tf, match[1])

		if err != nil {
			log.Fatal(err.Error())
		}

		year, week := dt.ISOWeek()

		//fmt.Println(line, "=>", dt, title, year, week)

		if year == current.Year && week == current.Week {
			current.Append(dt, title)
		} else {
			current.Print()
			current.New(year, week, dt)
		}
	}

	current.Print()
}

type WeekReport struct {
	Year int
	Week int
	Days []*DayReport
}

func (w *WeekReport) New(year, week int, start time.Time) {
	w.Year = year
	w.Week = week
	w.Days = []*DayReport{NewDayReport(start)}
}

func (w *WeekReport) Append(dt time.Time, title string) {
	work := !isStarred(title)
	day := w.lastDay()

	if day.Day == dt.Weekday() {
		day.Spans = append(day.Spans, Span{dt, work})
	} else {
		w.Days = append(w.Days, NewDayReport(dt))
	}
}

func (w *WeekReport) lastDay() *DayReport {
	days := w.Days
	return days[len(days)-1]
}

func (w *WeekReport) Print() {
	if len(w.Days) < 1 {
		return
	}
	fmt.Printf("%04d week %02d:\n", w.Year, w.Week)
	var days int
	var worked time.Duration
	for _, day := range w.Days {
		day.Print()
		days++
		worked += day.Worked()
	}
	expected := 7 * time.Hour * time.Duration(days)
	overtime := worked - expected
	daily := worked / time.Duration(days)
	fmt.Printf("   daily: %s\n", daily)
	fmt.Printf("  worked: %s\n", worked)
	fmt.Printf("  expect: %s\n", expected)
	if overtime > 0 {
		fmt.Printf("    over: %s\n", overtime)
	} else {
		fmt.Printf("   under: %s\n", -overtime)
	}
}

type DayReport struct {
	Day   time.Weekday
	Start time.Time
	Spans []Span
}

func NewDayReport(start time.Time) *DayReport {
	return &DayReport{
		Day:   start.Weekday(),
		Start: start,
	}
}

func (d *DayReport) Print() {
	fmt.Printf("  %s: %s\n", d.Start.Format("2006-01-02"), d.Worked())
}

func (d *DayReport) Worked() time.Duration {
	var worked time.Duration
	start := d.Start
	for _, span := range d.Spans {
		if span.Work {
			worked += span.End.Sub(start)
		}
		start = span.End
	}
	return worked
}

type Span struct {
	End  time.Time
	Work bool
}

func isStarred(title string) bool {
	_, found := strings.CutSuffix(title, "**")
	return found
}
