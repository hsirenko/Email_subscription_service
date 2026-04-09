package domain_test

import (
	"testing"

	"email-subscription-service/internal/domain"
)

func TestParseRepo(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    domain.Repo
		wantErr error
	}{
		{name: "ok", in: "golang/go", want: domain.Repo{Owner: "golang", Name: "go"}},
		{name: "trim", in: "  golang/go  ", want: domain.Repo{Owner: "golang", Name: "go"}},
		{name: "no slash", in: "bad", wantErr: domain.ErrInvalidRepoFormat},
		{name: "empty owner", in: "/go", wantErr: domain.ErrInvalidRepoFormat},
		{name: "empty name", in: "golang/", wantErr: domain.ErrInvalidRepoFormat},
		{name: "extra slash", in: "a/b/c", wantErr: domain.ErrInvalidRepoFormat},
		{name: "empty", in: "", wantErr: domain.ErrInvalidRepoFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseRepo(tt.in)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Fatalf("repo = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	if err := domain.ValidateEmail(""); err != domain.ErrInvalidEmail {
		t.Fatalf("empty: %v", err)
	}
	if err := domain.ValidateEmail("not-an-email"); err != domain.ErrInvalidEmail {
		t.Fatalf("bad: %v", err)
	}
	if err := domain.ValidateEmail("ok@example.com"); err != nil {
		t.Fatalf("valid: %v", err)
	}
}