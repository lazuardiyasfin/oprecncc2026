package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"net/http"
)

const (
	replaceChars      = `!@$&*`
	sepChars          = `_-., `
	otherSpecialChars = `"#%'()+/:;<=>?[\]^{|}~`
	lowerChars        = `abcdefghijklmnopqrstuvwxyz`
	upperChars        = `ABCDEFGHIJKLMNOPQRSTUVWXYZ`
	digitsChars       = `0123456789`
)

func getBase(password string) int {
	chars := map[rune]struct{}{}
	for _, char := range password {
		chars[char] = struct{}{}
	}

	hasReplace := false
	hasSep := false
	hasOtherSpecial := false
	hasLower := false
	hasUpper := false
	hasDigits := false
	base := 0

	for char := range chars {
		if strings.ContainsRune(replaceChars, char) {
			hasReplace = true
		} else if strings.ContainsRune(sepChars, char) {
			hasSep = true
		} else if strings.ContainsRune(otherSpecialChars, char) {
			hasOtherSpecial = true
		} else if strings.ContainsRune(lowerChars, char) {
			hasLower = true
		} else if strings.ContainsRune(upperChars, char) {
			hasUpper = true
		} else if strings.ContainsRune(digitsChars, char) {
			hasDigits = true
		} else {
			base++
		}
	}

	if hasReplace {
		base += len(replaceChars)
	}
	if hasSep {
		base += len(sepChars)
	}
	if hasOtherSpecial {
		base += len(otherSpecialChars)
	}
	if hasLower {
		base += len(lowerChars)
	}
	if hasUpper {
		base += len(upperChars)
	}
	if hasDigits {
		base += len(digitsChars)
	}

	return base
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<form method="POST">Password: <input type="text" name="password"><input type="submit" value="Check"></form>`)
		return
	}

	if r.Method == http.MethodPost {
		pass := r.FormValue("password")
		base := getBase(pass)
		length := len(pass)

		entropy := float64(length) * math.Log2(float64(base))

		var strength string
		if entropy < 40 {
			strength = "Very Weak"
		} else if entropy < 60 {
			strength = "Weak"
		} else if entropy < 80 {
			strength = "Medium"
		} else if entropy < 100 {
			strength = "Strong"
		} else {
			strength = "Very Strong"
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<p>Password: %s</p><p>Strength: %s</p><a href='/'>Back</a>", pass, strength)
	}
}

func main()  {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":5000", nil))
}