package utils

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"slices"
	"time"
)

var BaseXtermAnsiColorNames = []string{
	"black", // 0
	"maroon",
	"green",
	"olive",
	"navy",
	"purple",
	"teal",
	"silver", // 7
	"gray",   // 0 bright
	"red",
	"lime",
	"yellow",
	"blue",
	"fuchsia",
	"aqua",
	"white", // 7 bright
}

func GenerateRandomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
func GenerateRandomColor() string {
	index := rand.Intn(len(BaseXtermAnsiColorNames))
	return BaseXtermAnsiColorNames[index]
}

func Contains(s []string, color string) bool {
	return slices.Contains(s, color)
}

func FormatPrettyTime(unixMicro int64) string {
	t := time.UnixMicro(unixMicro)
	now := time.Now()
	year, month, day := t.Date()
	nowYear, nowMonth, nowDay := now.Date()

	timePart := t.Format("15:04")

	if year == nowYear && month == nowMonth && day == nowDay {
		return fmt.Sprintf("Today %s", timePart)
	}

	yesterday := now.AddDate(0, 0, -1)
	if year == yesterday.Year() && month == yesterday.Month() && day == yesterday.Day() {
		return fmt.Sprintf("Yesterday %s", timePart)
	}

	if year == nowYear {
		return fmt.Sprintf("%s %d %s", t.Format("Jan"), day, timePart)
	}

	return fmt.Sprintf("%d %s %02d %s", year, t.Format("Jan"), day, timePart)
}
