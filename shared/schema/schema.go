// Package schema veritabani semasini tasir. Butun ifadeler tekrar
// calistirilabilir; servisler acilista uygular, boylece var olan bir
// veritabani da yeni tablolara kavusur.
package schema

import _ "embed"

//go:embed schema.sql
var SQL string

// Ayni anda acilan servisler semayi ust uste uygulamasin diye alinan kilit.
const LockKey int64 = 7426001
