//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"threat-service/internal/repository/postgres"
	"threat-service/internal/transport"
	"threat-service/internal/usecase"
)

const postgresImage = "postgres:17.5-bookworm"

func TestThreatAPIWithPostgreSQL(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker не установлен: интеграционный тест пропущен")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	databaseURL := startPostgreSQL(t, ctx)
	pool := waitForPostgreSQL(t, ctx, databaseURL)
	defer pool.Close()
	applyMigrations(t, ctx, pool)

	threatService := usecase.NewThreatService(postgres.NewThreatRepository(pool), usecase.SystemClock{})
	server := httptest.NewServer(transport.NewHTTPHandler(threatService, pool, transport.NewRateLimiter(100, time.Minute)))
	defer server.Close()

	assertStatus(t, http.MethodGet, server.URL+"/health", "", http.StatusOK)

	threatMD5 := "d41d8cd98f00b204e9800998ecf8427e"
	createBody := `{
        "md5":"d41d8cd98f00b204e9800998ecf8427e",
        "name":"Trojan.Win32.Generic",
        "type":"trojan",
        "severity":"high",
        "description":"Вредоносная программа.",
        "categories":["banker","stealer"],
        "firstSeenAt":"2026-10-15T10:00:00Z"
    }`
	created := assertStatus(t, http.MethodPost, server.URL+"/info", createBody, http.StatusCreated)
	if created["md5"] != threatMD5 {
		t.Fatalf("md5 в ответе = %v, нужен %s", created["md5"], threatMD5)
	}
	if created["createdAt"] == nil {
		t.Fatal("в ответе на создание отсутствует createdAt")
	}

	assertStatus(t, http.MethodPost, server.URL+"/info", createBody, http.StatusConflict)
	loaded := assertStatus(t, http.MethodGet, server.URL+"/info/"+threatMD5, "", http.StatusOK)
	if loaded["name"] != "Trojan.Win32.Generic" {
		t.Fatalf("name в ответе = %v", loaded["name"])
	}

	updateBody := `{
        "name":"Trojan.Win32.Banker",
        "type":"trojan",
        "severity":"critical",
        "description":"Обновлённая классификация.",
        "categories":["banker","credential-theft"],
        "firstSeenAt":"2026-10-15T10:00:00Z"
    }`
	updated := assertStatus(t, http.MethodPut, server.URL+"/info/"+threatMD5, updateBody, http.StatusOK)
	if updated["name"] != "Trojan.Win32.Banker" || updated["severity"] != "critical" {
		t.Fatalf("обновлённая угроза = %#v", updated)
	}

	assertStatus(t, http.MethodGet, server.URL+"/info/not-an-md5", "", http.StatusBadRequest)
	assertStatus(t, http.MethodGet, server.URL+"/info/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "", http.StatusNotFound)
}

func startPostgreSQL(t *testing.T, ctx context.Context) string {
	t.Helper()

	containerName := fmt.Sprintf("threat-service-integration-%d", time.Now().UnixNano())
	command := exec.CommandContext(
		ctx,
		"docker", "run", "--detach", "--rm", "--name", containerName,
		"--env", "POSTGRES_DB=threat_service_test",
		"--env", "POSTGRES_USER=threat_service",
		"--env", "POSTGRES_PASSWORD=threat_service",
		"--publish", "127.0.0.1::5432",
		postgresImage,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("не удалось запустить %s: %v\n%s", postgresImage, err, output)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "--force", containerName).Run()
	})

	portCommand := exec.CommandContext(ctx, "docker", "port", containerName, "5432/tcp")
	portOutput, err := portCommand.Output()
	if err != nil {
		t.Fatalf("не удалось определить порт PostgreSQL: %v", err)
	}
	_, port, err := net.SplitHostPort(strings.TrimSpace(string(portOutput)))
	if err != nil {
		t.Fatalf("не удалось разобрать порт PostgreSQL: %v", err)
	}

	return fmt.Sprintf("postgres://threat_service:threat_service@127.0.0.1:%s/threat_service_test?sslmode=disable", port)
}

func waitForPostgreSQL(t *testing.T, ctx context.Context, databaseURL string) *pgxpool.Pool {
	t.Helper()

	for {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				return pool
			}
			pool.Close()
		}
		if ctx.Err() != nil {
			t.Fatalf("PostgreSQL не стал доступен: %v", ctx.Err())
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func applyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("не удалось определить путь к интеграционному тесту")
	}
	pattern := filepath.Join(filepath.Dir(sourceFile), "..", "..", "migrations", "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("не удалось найти миграции: %v", err)
	}
	sort.Strings(files)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("не удалось прочитать миграцию %s: %v", file, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			t.Fatalf("не удалось применить миграцию %s: %v", filepath.Base(file), err)
		}
	}
}

func assertStatus(t *testing.T, method, url, body string, expectedStatus int) map[string]any {
	t.Helper()

	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("не удалось создать запрос: %v", err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("запрос %s %s завершился с ошибкой: %v", method, url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != expectedStatus {
		t.Fatalf("%s %s: статус = %d, нужен %d", method, url, response.StatusCode, expectedStatus)
	}

	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("не удалось разобрать JSON-ответ: %v", err)
	}
	return result
}
