package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/service"
	_ "github.com/go-sql-driver/mysql"
)

type cityJob struct {
	ID          uint64
	CountryCode string
	Name        string
	Latitude    float64
	Longitude   float64
	TimezoneID  string
}

type cityResult struct {
	City    cityJob
	Failure string
}

type countryStat struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Scanned int64  `json:"scanned"`
	Passed  int64  `json:"passed"`
	Failed  int64  `json:"failed"`
}

type scanReport struct {
	FixedUTC               string        `json:"fixed_utc"`
	StartedAt              string        `json:"started_at"`
	FinishedAt             string        `json:"finished_at"`
	ElapsedSeconds         float64       `json:"elapsed_seconds"`
	Workers                int           `json:"workers"`
	CountryCount           int           `json:"country_count"`
	CountriesWithCities    int           `json:"countries_with_cities"`
	CountriesWithoutCities int           `json:"countries_without_cities"`
	UncoveredCountryCodes  []string      `json:"uncovered_country_codes"`
	DatabaseCities         int64         `json:"database_city_count"`
	Scanned                int64         `json:"scanned"`
	Passed                 int64         `json:"passed"`
	Failed                 int64         `json:"failed"`
	RatePerSecond          float64       `json:"rate_per_second"`
	Overall                string        `json:"overall"`
	Countries              []countryStat `json:"countries"`
	FailureCSV             string        `json:"failure_csv"`
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func main() {
	workers := flag.Int("workers", 8, "number of concurrent chart workers")
	limit := flag.Int("limit", 0, "maximum cities to scan; zero scans all")
	fixedUTCText := flag.String("utc", "2026-08-26T06:30:00Z", "fixed instant in RFC3339")
	outputDirectory := flag.String("output", "tmp/acceptance", "report output directory")
	flag.Parse()
	if *workers < 1 || *workers > 64 {
		fatal(errors.New("workers must be between 1 and 64"))
	}
	fixedUTC, err := time.Parse(time.RFC3339, *fixedUTCText)
	if err != nil {
		fatal(fmt.Errorf("parse fixed UTC: %w", err))
	}
	fixedUTC = fixedUTC.UTC()
	logger.Init("error")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		env("DB_USER", "fatelumen"), env("DB_PASSWORD", "fatelumen123"), env("DB_HOST", "127.0.0.1"), env("DB_PORT", "3307"), env("DB_NAME", "fatelumen"))
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(2 * time.Minute)
	if err := db.Ping(); err != nil {
		fatal(fmt.Errorf("connect geo database: %w", err))
	}

	countries, err := loadCountries(db)
	if err != nil {
		fatal(err)
	}
	var databaseCities int64
	if err := db.QueryRow("SELECT COUNT(*) FROM geo_cities").Scan(&databaseCities); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*outputDirectory, 0o755); err != nil {
		fatal(err)
	}
	stamp := time.Now().Format("20060102-150405")
	failurePath := filepath.Join(*outputDirectory, "birthchart-full-failures-"+stamp+".csv")
	failureFile, err := os.Create(failurePath)
	if err != nil {
		fatal(err)
	}
	failureWriter := csv.NewWriter(failureFile)
	_ = failureWriter.Write([]string{"geo_name_id", "country_code", "city", "latitude", "longitude", "timezone_id", "failure"})

	started := time.Now()
	jobs := make(chan cityJob, *workers*4)
	results := make(chan cityResult, *workers*4)
	var workerGroup sync.WaitGroup
	for i := 0; i < *workers; i++ {
		workerGroup.Add(1)
		go func() {
			defer workerGroup.Done()
			engine := birthchart.NewDefaultEngine()
			freeService := service.NewFreeChartService(engine)
			for job := range jobs {
				results <- validateCity(context.Background(), fixedUTC, countries, engine, freeService, job)
			}
		}()
	}
	producerErrors := make(chan error, 1)
	go func() {
		producerErrors <- produceCities(db, *limit, jobs)
		close(jobs)
	}()
	go func() {
		workerGroup.Wait()
		close(results)
	}()

	stats := make(map[string]*countryStat, len(countries))
	for code, name := range countries {
		stats[code] = &countryStat{Code: code, Name: name}
	}
	var scanned, passed, failed int64
	for result := range results {
		scanned++
		stat, ok := stats[result.City.CountryCode]
		if !ok {
			stat = &countryStat{Code: result.City.CountryCode}
			stats[result.City.CountryCode] = stat
		}
		stat.Scanned++
		if result.Failure == "" {
			passed++
			stat.Passed++
		} else {
			failed++
			stat.Failed++
			_ = failureWriter.Write([]string{strconv.FormatUint(result.City.ID, 10), result.City.CountryCode, result.City.Name, strconv.FormatFloat(result.City.Latitude, 'f', 7, 64), strconv.FormatFloat(result.City.Longitude, 'f', 7, 64), result.City.TimezoneID, result.Failure})
		}
		if scanned%10000 == 0 {
			elapsed := time.Since(started).Seconds()
			fmt.Printf("progress scanned=%d passed=%d failed=%d rate=%.1f/s\n", scanned, passed, failed, float64(scanned)/elapsed)
		}
	}
	producerErr := <-producerErrors
	failureWriter.Flush()
	closeErr := failureFile.Close()
	if producerErr != nil {
		fatal(producerErr)
	}
	if err := failureWriter.Error(); err != nil {
		fatal(err)
	}
	if closeErr != nil {
		fatal(closeErr)
	}

	finished := time.Now()
	elapsed := finished.Sub(started).Seconds()
	countryRows := make([]countryStat, 0, len(stats))
	uncoveredCountryCodes := make([]string, 0)
	for _, stat := range stats {
		countryRows = append(countryRows, *stat)
		if stat.Scanned == 0 {
			uncoveredCountryCodes = append(uncoveredCountryCodes, stat.Code)
		}
	}
	sort.Slice(countryRows, func(i, j int) bool { return countryRows[i].Code < countryRows[j].Code })
	sort.Strings(uncoveredCountryCodes)
	overall := "PASS"
	if failed > 0 || (*limit == 0 && scanned != databaseCities) {
		overall = "FAIL"
	}
	report := scanReport{FixedUTC: fixedUTC.Format(time.RFC3339), StartedAt: started.Format(time.RFC3339), FinishedAt: finished.Format(time.RFC3339), ElapsedSeconds: elapsed, Workers: *workers, CountryCount: len(countries), CountriesWithCities: len(countries) - len(uncoveredCountryCodes), CountriesWithoutCities: len(uncoveredCountryCodes), UncoveredCountryCodes: uncoveredCountryCodes, DatabaseCities: databaseCities, Scanned: scanned, Passed: passed, Failed: failed, RatePerSecond: float64(scanned) / elapsed, Overall: overall, Countries: countryRows, FailureCSV: failurePath}
	reportPath := filepath.Join(*outputDirectory, "birthchart-full-acceptance-"+stamp+".json")
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("full_scan overall=%s countries=%d covered_countries=%d skipped_countries=%d database_cities=%d scanned=%d passed=%d failed=%d elapsed=%.1fs rate=%.1f/s\n", overall, len(countries), report.CountriesWithCities, report.CountriesWithoutCities, databaseCities, scanned, passed, failed, elapsed, report.RatePerSecond)
	fmt.Printf("report=%s\nfailures=%s\n", reportPath, failurePath)
	if overall != "PASS" {
		os.Exit(1)
	}
}

func loadCountries(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query("SELECT code, name_en FROM geo_countries ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	countries := map[string]string{}
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			return nil, err
		}
		countries[code] = name
	}
	return countries, rows.Err()
}

func produceCities(db *sql.DB, limit int, jobs chan<- cityJob) error {
	const batchSize = 5000
	var lastID uint64
	produced := 0
	for {
		size := batchSize
		if limit > 0 && limit-produced < size {
			size = limit - produced
		}
		if size <= 0 {
			return nil
		}
		var batch []cityJob
		var err error
		for attempt := 1; attempt <= 3; attempt++ {
			batch, err = loadCityBatch(db, lastID, size)
			if err == nil {
				break
			}
			_ = db.Ping()
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
		if err != nil {
			return fmt.Errorf("load city batch after id %d: %w", lastID, err)
		}
		if len(batch) == 0 {
			return nil
		}
		for _, job := range batch {
			jobs <- job
		}
		produced += len(batch)
		lastID = batch[len(batch)-1].ID
		if len(batch) < size {
			return nil
		}
	}
}

func loadCityBatch(db *sql.DB, afterID uint64, size int) ([]cityJob, error) {
	rows, err := db.Query("SELECT geo_name_id, country_code, name_en, latitude, longitude, timezone_id FROM geo_cities WHERE geo_name_id > ? ORDER BY geo_name_id LIMIT ?", afterID, size)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	batch := make([]cityJob, 0, size)
	for rows.Next() {
		var job cityJob
		if err := rows.Scan(&job.ID, &job.CountryCode, &job.Name, &job.Latitude, &job.Longitude, &job.TimezoneID); err != nil {
			return nil, err
		}
		batch = append(batch, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return batch, nil
}

func validateCity(ctx context.Context, fixedUTC time.Time, countries map[string]string, engine birthchart.Engine, freeService *service.FreeChartService, city cityJob) cityResult {
	fail := func(err error) cityResult { return cityResult{City: city, Failure: err.Error()} }
	if _, ok := countries[city.CountryCode]; !ok {
		return fail(errors.New("country not found"))
	}
	if city.Latitude < -90 || city.Latitude > 90 || city.Longitude < -180 || city.Longitude > 180 || math.IsNaN(city.Latitude) || math.IsNaN(city.Longitude) || math.IsInf(city.Latitude, 0) || math.IsInf(city.Longitude, 0) {
		return fail(errors.New("invalid coordinates"))
	}
	location, err := time.LoadLocation(city.TimezoneID)
	if err != nil {
		return fail(fmt.Errorf("invalid IANA timezone: %w", err))
	}
	wall := fixedUTC.In(location)
	locationInput := birthchart.LocationInput{CountryCode: city.CountryCode, City: city.Name, PlaceID: strconv.FormatUint(city.ID, 10), DisplayName: city.Name, Latitude: city.Latitude, Longitude: city.Longitude, TimezoneID: city.TimezoneID, HasCoordinates: true}
	input := birthchart.Input{Gender: 1, CalendarType: 0, Year: wall.Year(), Month: int(wall.Month()), Day: wall.Day(), Hour: wall.Hour(), Minute: wall.Minute(), Location: locationInput}
	reportResult, err := engine.Calculate(ctx, input)
	if err != nil {
		return fail(fmt.Errorf("report engine: %w", err))
	}
	if !reportResult.Timezone.BirthUTC.Equal(fixedUTC) {
		return fail(fmt.Errorf("UTC roundtrip mismatch: got %s", reportResult.Timezone.BirthUTC.Format(time.RFC3339)))
	}
	if err := completePillars(reportResult); err != nil {
		return fail(err)
	}
	freeInput := service.FreeChartInput{Gender: input.Gender, CalendarType: input.CalendarType, Year: input.Year, Month: input.Month, Day: input.Day, Hour: input.Hour, Minute: input.Minute, Location: locationInput}
	freeResult, err := freeService.Calculate(ctx, freeInput)
	if err != nil {
		return fail(fmt.Errorf("free chart: %w", err))
	}
	reportHash := service.BuildChartHash(input.Gender, input.CalendarType, input.Year, input.Month, input.Day, input.Hour, input.Minute, false, reportResult)
	if freeResult.ChartHash != reportHash || !reflect.DeepEqual(freeResult.Pillars, reportResult.Chart.Pillars) || !reflect.DeepEqual(freeResult.TimeCalculation, reportResult.Chart.Meta.TimeCalculation) {
		return fail(errors.New("free/report result mismatch"))
	}
	return cityResult{City: city}
}

func completePillars(result *birthchart.Result) error {
	p := result.Chart.Pillars
	values := []string{p.Year.Stem, p.Year.Branch, p.Month.Stem, p.Month.Branch, p.Day.Stem, p.Day.Branch, p.Hour.Stem, p.Hour.Branch}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return errors.New("incomplete pillars")
		}
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "full birth chart acceptance failed:", err)
	os.Exit(1)
}
