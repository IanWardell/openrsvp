package database

import "testing"

func TestRewritePlaceholders(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no placeholders",
			in:   "SELECT * FROM events",
			want: "SELECT * FROM events",
		},
		{
			name: "single placeholder",
			in:   "SELECT * FROM events WHERE id = ?",
			want: "SELECT * FROM events WHERE id = $1",
		},
		{
			name: "two placeholders increment",
			in:   "INSERT INTO t (a, b) VALUES (?, ?)",
			want: "INSERT INTO t (a, b) VALUES ($1, $2)",
		},
		{
			name: "question mark inside single-quote literal preserved",
			in:   "SELECT * FROM t WHERE name = 'who?' AND id = ?",
			want: "SELECT * FROM t WHERE name = 'who?' AND id = $1",
		},
		{
			name: "question mark inside double-quote identifier preserved",
			in:   `SELECT "wat?col" FROM t WHERE id = ?`,
			want: `SELECT "wat?col" FROM t WHERE id = $1`,
		},
		{
			name: "question mark inside line comment preserved",
			in:   "SELECT id -- what? not a param\nFROM t WHERE id = ?",
			want: "SELECT id -- what? not a param\nFROM t WHERE id = $1",
		},
		{
			name: "question mark inside block comment preserved",
			in:   "SELECT id /* what? not a param */ FROM t WHERE id = ?",
			want: "SELECT id /* what? not a param */ FROM t WHERE id = $1",
		},
		{
			name: "doubled single-quote escape",
			in:   "SELECT * FROM t WHERE s = 'it''s a ?' AND id = ?",
			want: "SELECT * FROM t WHERE s = 'it''s a ?' AND id = $1",
		},
		{
			name: "doubled double-quote escape",
			in:   `SELECT "we""ird?col" FROM t WHERE a = ? AND b = ?`,
			want: `SELECT "we""ird?col" FROM t WHERE a = $1 AND b = $2`,
		},
		{
			name: "realistic multi-placeholder query",
			in:   "UPDATE events SET title = ?, show_headcount = ?, comments_enabled = ? WHERE id = ?",
			want: "UPDATE events SET title = $1, show_headcount = $2, comments_enabled = $3 WHERE id = $4",
		},
		{
			name: "dynamic IN clause",
			in:   "SELECT * FROM events WHERE id IN (?, ?, ?)",
			want: "SELECT * FROM events WHERE id IN ($1, $2, $3)",
		},
		{
			name: "placeholder after closing line comment",
			in:   "-- leading comment ?\nSELECT ? FROM t",
			want: "-- leading comment ?\nSELECT $1 FROM t",
		},
		{
			name: "placeholders around block comment",
			in:   "SELECT ? /* mid ? */ , ? FROM t",
			want: "SELECT $1 /* mid ? */ , $2 FROM t",
		},
		{
			name: "block comment terminator emitted intact",
			in:   "SELECT /* a */ ? FROM t",
			want: "SELECT /* a */ $1 FROM t",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewritePlaceholders(tt.in)
			if got != tt.want {
				t.Errorf("rewritePlaceholders(%q)\n  got  %q\n  want %q", tt.in, got, tt.want)
			}
		})
	}
}
