package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"fatelumen/backend/internal/config"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type localizedNames struct {
	EN, ZH, JA, KO string
	preferred      map[string]bool
}

type importStats struct {
	SourceCities, ImportedCities, InvalidCoordinates, MissingTimezone int
	MissingZH, MissingJA, MissingKO                                   int
	DroppedChinaWithoutZH                                             int
}

var chinaAdmin1ZH = map[string]string{
	"01": "安徽", "02": "浙江", "03": "江西", "04": "江苏", "05": "吉林",
	"06": "青海", "07": "福建", "08": "黑龙江", "09": "河南", "10": "河北",
	"11": "湖南", "12": "湖北", "13": "新疆", "14": "西藏", "15": "甘肃",
	"16": "广西", "18": "贵州", "19": "辽宁", "20": "内蒙古", "21": "宁夏",
	"22": "北京", "23": "上海", "24": "山西", "25": "山东", "26": "陕西",
	"28": "天津", "29": "云南", "30": "广东", "31": "海南", "32": "四川",
	"33": "重庆",
}

func main() {
	log := logger.Init("info")
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("load config failed", "err", err)
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		log.Fatal("connect database failed", "err", err)
	}
	if err := db.AutoMigrate(&model.GeoCountry{}, &model.GeoCity{}); err != nil {
		log.Fatal("migrate geo tables failed", "err", err)
	}
	var existing int64
	if err := db.Model(&model.GeoCountry{}).Count(&existing).Error; err != nil {
		log.Fatal("check geo data failed", "err", err)
	}
	if existing > 0 {
		log.Fatal("geo base data already exists; one-time import refused", "country_count", existing)
	}

	dir := "tmp/geonames"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	countries, countryIDs, err := readCountries(filepath.Join(dir, "countryInfo.txt"))
	if err != nil {
		log.Fatal("read countries failed", "err", err)
	}
	admin1, err := readAdmin1(filepath.Join(dir, "admin1CodesASCII.txt"))
	if err != nil {
		log.Fatal("read admin1 failed", "err", err)
	}
	cities, wanted, stats, err := readCities(filepath.Join(dir, "cities500.zip"), admin1)
	if err != nil {
		log.Fatal("read cities failed", "err", err)
	}
	for _, id := range countryIDs {
		wanted[id] = struct{}{}
	}
	names, err := readAlternateNames(filepath.Join(dir, "alternateNamesV2.zip"), wanted)
	if err != nil {
		log.Fatal("read localized names failed", "err", err)
	}

	for i := range countries {
		applyCountryNames(&countries[i], names[countryIDs[countries[i].Code]])
	}
	for i := range cities {
		applyCityNames(&cities[i], names[cities[i].GeoNameID], &stats)
	}
	cities = keepLocalizedChinaCities(cities, &stats)
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(countries, 500).Error; err != nil {
			return err
		}
		return tx.CreateInBatches(cities, 1000).Error
	})
	if err != nil {
		log.Fatal("import geo data failed", "err", err)
	}
	stats.ImportedCities = len(cities)
	log.Info("geo base data imported", "countries", len(countries), "source_cities", stats.SourceCities, "cities", stats.ImportedCities, "invalid_coordinates", stats.InvalidCoordinates, "missing_timezone", stats.MissingTimezone, "missing_zh", stats.MissingZH, "missing_ja", stats.MissingJA, "missing_ko", stats.MissingKO, "dropped_china_without_zh", stats.DroppedChinaWithoutZH)
}

func readCountries(path string) ([]model.GeoCountry, map[string]uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var rows []model.GeoCountry
	ids := map[string]uint64{}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Split(line, "\t")
		if len(p) < 17 {
			continue
		}
		id, err := strconv.ParseUint(p[16], 10, 64)
		if err != nil {
			continue
		}
		code, name := p[0], p[4]
		rows = append(rows, model.GeoCountry{Code: code, NameEN: name, NameZH: name, NameJA: name, NameKO: name})
		ids[code] = id
	}
	return rows, ids, s.Err()
}

func readAdmin1(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		p := strings.Split(s.Text(), "\t")
		if len(p) >= 2 {
			rows[p[0]] = p[1]
		}
	}
	return rows, s.Err()
}

func zipReader(path, filename string) (io.ReadCloser, *zip.ReadCloser, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, nil, err
	}
	if len(z.File) == 0 {
		z.Close()
		return nil, nil, fmt.Errorf("empty zip: %s", path)
	}
	var selected *zip.File
	for _, item := range z.File {
		if strings.EqualFold(item.Name, filename) {
			selected = item
			break
		}
	}
	if selected == nil {
		z.Close()
		return nil, nil, fmt.Errorf("%s not found in %s", filename, path)
	}
	r, err := selected.Open()
	if err != nil {
		z.Close()
		return nil, nil, err
	}
	return r, z, nil
}

func readCities(path string, admin1 map[string]string) ([]model.GeoCity, map[uint64]struct{}, importStats, error) {
	r, z, err := zipReader(path, "cities500.txt")
	if err != nil {
		return nil, nil, importStats{}, err
	}
	defer z.Close()
	defer r.Close()
	rows := make([]model.GeoCity, 0, 190000)
	wanted := make(map[uint64]struct{}, 190000)
	stats := importStats{}
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		stats.SourceCities++
		p := strings.Split(s.Text(), "\t")
		if len(p) < 19 {
			continue
		}
		id, e0 := strconv.ParseUint(p[0], 10, 64)
		lat, e1 := strconv.ParseFloat(p[4], 64)
		lng, e2 := strconv.ParseFloat(p[5], 64)
		if e0 != nil || e1 != nil || e2 != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			stats.InvalidCoordinates++
			continue
		}
		if strings.TrimSpace(p[17]) == "" {
			stats.MissingTimezone++
			continue
		}
		code := p[8]
		adminCode := p[10]
		rows = append(rows, model.GeoCity{GeoNameID: id, CountryCode: code, Admin1Code: adminCode, Admin1Name: admin1[code+"."+adminCode], NameEN: p[1], NameZH: p[1], NameJA: p[1], NameKO: p[1], Latitude: lat, Longitude: lng, TimezoneID: p[17]})
		wanted[id] = struct{}{}
	}
	return rows, wanted, stats, s.Err()
}

func readAlternateNames(path string, wanted map[uint64]struct{}) (map[uint64]*localizedNames, error) {
	r, z, err := zipReader(path, "alternateNamesV2.txt")
	if err != nil {
		return nil, err
	}
	defer z.Close()
	defer r.Close()
	rows := make(map[uint64]*localizedNames, len(wanted))
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		p := strings.Split(s.Text(), "\t")
		if len(p) < 8 || p[6] == "1" || p[7] == "1" {
			continue
		}
		id, err := strconv.ParseUint(p[1], 10, 64)
		if err != nil {
			continue
		}
		if _, ok := wanted[id]; !ok {
			continue
		}
		lang := normalizeLanguage(p[2])
		if lang == "" {
			continue
		}
		n := rows[id]
		if n == nil {
			n = &localizedNames{preferred: map[string]bool{}}
			rows[id] = n
		}
		preferred := p[4] == "1"
		target := nameTarget(n, lang)
		if *target == "" || preferred && !n.preferred[lang] {
			*target = p[3]
			n.preferred[lang] = preferred
		}
	}
	return rows, s.Err()
}

func normalizeLanguage(lang string) string {
	lang = strings.ToLower(lang)
	if lang == "en" || lang == "ja" || lang == "ko" {
		return lang
	}
	if lang == "zh" || strings.HasPrefix(lang, "zh-") {
		return "zh"
	}
	return ""
}
func nameTarget(n *localizedNames, lang string) *string {
	switch lang {
	case "zh":
		return &n.ZH
	case "ja":
		return &n.JA
	case "ko":
		return &n.KO
	default:
		return &n.EN
	}
}
func applyCountryNames(row *model.GeoCountry, n *localizedNames) {
	if n == nil {
		return
	}
	if n.EN != "" {
		row.NameEN = n.EN
	}
	if n.ZH != "" {
		row.NameZH = n.ZH
	}
	if n.JA != "" {
		row.NameJA = n.JA
	}
	if n.KO != "" {
		row.NameKO = n.KO
	}
}
func applyCityNames(row *model.GeoCity, n *localizedNames, stats *importStats) {
	if n != nil && n.EN != "" {
		row.NameEN = n.EN
	}
	if n != nil && n.ZH != "" {
		row.NameZH = n.ZH
	} else {
		row.NameZH = row.NameEN
		stats.MissingZH++
	}
	if n != nil && n.JA != "" {
		row.NameJA = n.JA
	} else {
		row.NameJA = row.NameEN
		stats.MissingJA++
	}
	if n != nil && n.KO != "" {
		row.NameKO = n.KO
	} else {
		row.NameKO = row.NameEN
		stats.MissingKO++
	}
}

func keepLocalizedChinaCities(rows []model.GeoCity, stats *importStats) []model.GeoCity {
	result := make([]model.GeoCity, 0, len(rows))
	for i := range rows {
		row := rows[i]
		if row.CountryCode == "CN" {
			if !containsHan(row.NameZH) {
				stats.DroppedChinaWithoutZH++
				continue
			}
			if adminName := chinaAdmin1ZH[row.Admin1Code]; adminName != "" {
				row.Admin1Name = adminName
			}
		}
		result = append(result, row)
	}
	return result
}

func containsHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
