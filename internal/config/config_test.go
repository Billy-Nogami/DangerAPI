package config

import "testing"

func TestDatabaseTargetDoesNotExposeCredentials(t *testing.T) {
	t.Parallel()

	actual := DatabaseTarget("postgres://service:secret@db:5432/threat_service?sslmode=disable")
	if actual != "db:5432/threat_service" {
		t.Fatalf("адрес БД = %q, нужен %q", actual, "db:5432/threat_service")
	}
}

func TestDefaultPublicBaseURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		httpAddress string
		want        string
	}{
		{
			name:        "все сетевые интерфейсы",
			httpAddress: ":8080",
			want:        "http://localhost:8080",
		},
		{
			name:        "локальный интерфейс",
			httpAddress: "127.0.0.1:9090",
			want:        "http://127.0.0.1:9090",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if actual := defaultPublicBaseURL(testCase.httpAddress); actual != testCase.want {
				t.Fatalf("внешний адрес = %q, нужен %q", actual, testCase.want)
			}
		})
	}
}
