package main

import (
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	_ "modernc.org/sqlite"
)

type Config struct {
	Port          string            `toml:"port"`
	DBPath        string            `toml:"db_path"`
	APIPath       string            `toml:"api_path"`
	TLSCert       string            `toml:"tls_cert"`
	TLSKey        string            `toml:"tls_key"`
	CosmeticsPath string            `toml:"cosmetics_path"`
	Users         map[string]string `toml:"users"`
	Static        []StaticDir       `toml:"static"`
}

type StaticDir struct {
	URLPath string `toml:"url_path"`
	FSPath  string `toml:"fs_path"`
}

var (
	cfg    Config
	db     *sql.DB
	gzPool = sync.Pool{New: func() interface{} { return gzip.NewWriter(nil) }}
)

func main() {
	configPath := "japro-stats.toml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		log.Fatalf("config: %v", err)
	}

	var err error
	dsn := fmt.Sprintf("file:%s?mode=ro", cfg.DBPath)
	db, err = sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		log.Fatalf("pragma: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	mux := http.NewServeMux()

	apiPath := cfg.APIPath
	if apiPath == "" {
		apiPath = "/update"
	}
	mux.HandleFunc(apiPath, handleAPI)

	for _, s := range cfg.Static {
		urlPath := s.URLPath
		fsPath := s.FSPath
		if !strings.HasSuffix(urlPath, "/") {
			urlPath += "/"
		}
		mux.Handle(urlPath, http.StripPrefix(urlPath, downloadHandler(http.FileServer(http.Dir(fsPath)))))
	}

	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("listening on %s", cfg.Port)
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		log.Fatal(srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey))
	} else {
		log.Fatal(srv.ListenAndServe())
	}
}

func downloadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".cfg") {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", "attachment")
		}
		next.ServeHTTP(w, r)
	})
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)

	username := sanitize(r.FormValue("username"))
	password := sanitize(r.FormValue("password"))
	qtype := sanitize(r.FormValue("type"))

	lastUpdateStr := r.FormValue("last_update")
	if lastUpdateStr == "" {
		lastUpdateStr = "0"
	}
	lastUpdate, err := strconv.ParseInt(sanitize(lastUpdateStr), 10, 64)
	if err != nil {
		http.Error(w, "invalid last_update", http.StatusBadRequest)
		return
	}

	if expected, ok := cfg.Users[username]; !ok || expected != password {
		http.Error(w, "Bad credentials", http.StatusUnauthorized)
		return
	}

	var result [][]interface{}
	switch qtype {
	case "check_password":
		player := sanitize(r.FormValue("player"))
		playerPass := sanitizeN(r.FormValue("player_password"), 64)
		var ok bool
		ok, err = checkPlayerPassword(player, playerPass)
		if err == nil {
			result = [][]interface{}{{ok}}
		}
	case "cosmetics":
		result, err = readCosmetics()
	default:
		result, err = runQuery(qtype, lastUpdate)
	}

	if err != nil {
		log.Printf("query %q: %v", qtype, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() { gz.Close(); gzPool.Put(gz) }()
		json.NewEncoder(gz).Encode(result)
	} else {
		json.NewEncoder(w).Encode(result)
	}
}

func sanitize(s string) string {
	return sanitizeN(s, 16)
}

func sanitizeN(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		s = s[:n]
	}
	return s
}

func checkPlayerPassword(player, pass string) (bool, error) {
	var stored string
	err := db.QueryRow(`SELECT password FROM LocalAccount WHERE username = ?`, player).Scan(&stored)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return stored == pass, nil
}

func readCosmetics() ([][]interface{}, error) {
	data, err := os.ReadFile(cfg.CosmeticsPath)
	if err != nil {
		return nil, err
	}
	var out [][]interface{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ";", 4)
		if len(parts) != 4 {
			continue
		}
		bit, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		coursename := strings.TrimSpace(parts[1])
		style, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			continue
		}
		ms, err := strconv.Atoi(strings.TrimSpace(parts[3]))
		if err != nil {
			continue
		}
		out = append(out, []interface{}{bit, coursename, style, ms})
	}
	return out, nil
}

func runQuery(qtype string, lastUpdate int64) ([][]interface{}, error) {
	switch qtype {
	case "races":
		return scan(
			`SELECT username, coursename, style, season, duration_ms, topspeed, average,
			        end_time, rank, entries, season_rank, season_entries, invalid
			 FROM LocalRun WHERE last_update > ? ORDER BY end_time DESC`,
			lastUpdate,
		)
	case "race_demos":
		return scan(
			`SELECT username, coursename, style, MIN(duration_ms)
			 FROM LocalRun WHERE rank = 1 AND end_time > ?
			 GROUP BY username, coursename, style
			 ORDER BY end_time DESC`,
			lastUpdate,
		)
	case "duels":
		return scan(
			`SELECT winner, loser, type, duration, winner_hp, winner_shield,
			        end_time, winner_elo, loser_elo, odds
			 FROM LocalDuel WHERE end_time > ? ORDER BY end_time DESC`,
			lastUpdate,
		)
	case "accounts":
		return scan(
			`SELECT username, kills, deaths, suicides, captures, returns, lastlogin, created, master
			 FROM LocalAccount WHERE lastlogin > ? ORDER BY lastlogin DESC`,
			lastUpdate,
		)
	case "teams":
		return scan(`SELECT name, tag, longname, flags FROM LocalTeam`)
	case "team_accounts":
		return scan(`SELECT team, account, flags FROM LocalTeamAccount`)
	case "race_alerts":
		return scan(
			`SELECT r.username, r.coursename, r.style, r.duration_ms, r.rank, r.entries,
			        COALESCE(fs.second_time - fs.first_time, 0) as delta
			 FROM LocalRun r
			 LEFT JOIN (
			     SELECT coursename, style,
			         MIN(CASE WHEN rank = 1 THEN duration_ms END) as first_time,
			         MIN(CASE WHEN rank = 2 THEN duration_ms END) as second_time
			     FROM LocalRun WHERE rank IN (1, 2)
			     GROUP BY coursename, style
			 ) fs ON r.coursename = fs.coursename AND r.style = fs.style
			 WHERE r.rank = 1 AND r.end_time > ?
			 ORDER BY r.end_time ASC`,
			lastUpdate,
		)
	default:
		return nil, fmt.Errorf("unknown type %q", qtype)
	}
}

func scan(query string, args ...interface{}) ([][]interface{}, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	n := len(cols)

	var out [][]interface{}
	for rows.Next() {
		vals := make([]interface{}, n)
		ptrs := make([]interface{}, n)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		out = append(out, vals)
	}
	return out, rows.Err()
}
