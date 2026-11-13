// Package namematch decides whether two spellings could name the same person.
//
// University records usually hold an Arabic name while ORCID and OpenAlex hold a
// Latin one, so names are reduced to a consonant skeleton shared by both scripts.
// The comparison is a sanity check, not proof of identity: it stops a stranger's
// record from being imported, and a false negative only sends a case to staff review.
package namematch

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const (
	// minStrongSkeleton is how many skeleton letters full-token matches must cover,
	// so "Ali A." cannot pass on a single common letter.
	minStrongSkeleton = 3
	// minFuzzySkeleton is the shortest skeleton that tolerates a one-letter slip.
	minFuzzySkeleton = 4
)

var honorifics = setOf(
	"dr", "prof", "professor", "mr", "mrs", "ms", "sir", "phd", "eng", "assoc", "asst", "assist", "lecturer",
	"د", "ا", "م", "دكتور", "الدكتور", "دكتوره", "الدكتوره", "استاذ", "الاستاذ", "استاذه", "الاستاذه",
	"مهندس", "المهندس", "مساعد", "مدرس", "المدرس",
)

// Words joining a name part to the next one: عبد الرحمن is one name, however it is spaced.
var compoundHeads = setOf("abd", "abdul", "abdel", "abdal", "abdol", "abdu", "abu", "abo", "عبد", "ابو")

var particles = setOf("al", "el", "ad", "ul", "bin", "ben", "ibn", "bint", "de", "van", "von", "بن", "ابن", "بنت")

// Latin digraphs are read before single letters.
var latinDigraphs = map[string]rune{"sh": 's', "kh": 'k', "gh": 'k', "th": 't', "dh": 'd', "ph": 'f', "ch": 'k', "ck": 'k'}

// Sounds that Latin transliteration of Arabic spells inconsistently share a class;
// long vowels and weak letters have no class because transliteration varies most there.
var skeletonClass = map[rune]rune{
	'b': 'b', 'p': 'b', 'c': 'k', 'd': 'd', 'f': 'f', 'v': 'f', 'g': 'k', 'h': 'h', 'j': 'j', 'k': 'k', 'l': 'l',
	'm': 'm', 'n': 'n', 'q': 'k', 'r': 'r', 's': 's', 't': 't', 'x': 'k', 'z': 'd',

	'ب': 'b', 'پ': 'b', 'ت': 't', 'ث': 't', 'ط': 't', 'ج': 'j', 'چ': 'j', 'ژ': 'j', 'ح': 'h', 'ه': 'h', 'ھ': 'h',
	'خ': 'k', 'غ': 'k', 'ق': 'k', 'ك': 'k', 'ک': 'k', 'گ': 'k', 'د': 'd', 'ذ': 'd', 'ز': 'd', 'ض': 'd', 'ظ': 'd',
	'ر': 'r', 'س': 's', 'ش': 's', 'ص': 's', 'ف': 'f', 'ڤ': 'f', 'ل': 'l', 'م': 'm', 'ن': 'n',
}

var droppedLetters = setOf(
	"a", "e", "i", "o", "u", "y", "w",
	"ا", "و", "ي", "ى", "ی", "ء", "ئ", "ؤ", "ع", "ة",
)

func setOf(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func has(set map[string]struct{}, value string) bool {
	_, ok := set[value]
	return ok
}

// token is one name part: its skeleton spellings, and its initial when abbreviated.
type token struct {
	forms   []string
	initial string
}

func (t token) strength() int {
	strongest := 0
	for _, form := range t.forms {
		strongest = max(strongest, len(form))
	}
	return strongest
}

// words folds accents, Arabic diacritics and hamza carriers away and splits on anything else.
func words(name string) []string {
	var folded strings.Builder
	for _, r := range norm.NFKD.String(strings.ToLower(name)) {
		switch {
		case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Cf, r), strings.ContainsRune("'’`ʿʾ", r):
		case unicode.IsLetter(r):
			folded.WriteRune(r)
		default:
			folded.WriteRune(' ')
		}
	}
	return strings.Fields(folded.String())
}

// joinCompounds glues "abd"/"abu" to what follows, including any "al" between them.
func joinCompounds(parts []string) []string {
	joined := make([]string, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if has(compoundHeads, part) {
			for i+1 < len(parts) {
				i++
				part += parts[i]
				if !has(particles, parts[i]) {
					break
				}
			}
		}
		joined = append(joined, part)
	}
	return joined
}

func stripPrefix(part string) (string, bool) {
	for _, prefix := range []string{"ال", "al", "el"} {
		rest, found := strings.CutPrefix(part, prefix)
		if found && len([]rune(rest)) >= 3 {
			return rest, true
		}
	}
	return "", false
}

// skeleton reduces one word to its consonant classes.
func skeleton(word string) string {
	runes := []rune(word)
	letters := make([]rune, 0, len(runes))
	for i := 0; i < len(runes); i++ {
		class, ok := rune(0), false
		if i+1 < len(runes) {
			if class, ok = latinDigraphs[string(runes[i:i+2])]; ok {
				i++
			}
		}
		if !ok {
			if has(droppedLetters, string(runes[i])) {
				continue
			}
			if class, ok = skeletonClass[runes[i]]; !ok {
				class = runes[i] // other scripts stay themselves and match only themselves
			}
		}
		if len(letters) == 0 || letters[len(letters)-1] != class {
			letters = append(letters, class)
		}
	}
	// A final h is often silent: Fatimah / فاطمة, Abdullah / عبدالله.
	if n := len(letters); n > 1 && letters[n-1] == 'h' {
		letters = letters[:n-1]
	}
	return string(letters)
}

func tokenize(name string) []token {
	tokens := []token{}
	for _, part := range joinCompounds(words(name)) {
		if has(honorifics, part) || has(particles, part) {
			continue
		}
		if runes := []rune(part); len(runes) == 1 {
			tokens = append(tokens, token{initial: skeleton(part), forms: nil})
			continue
		}
		forms := []string{skeleton(part)}
		if rest, ok := stripPrefix(part); ok {
			forms = append(forms, skeleton(rest))
		}
		tokens = append(tokens, token{forms: forms})
	}
	return tokens
}

func editDistanceAtMostOne(a, b string) bool {
	ra, rb := []rune(a), []rune(b)
	if len(ra) > len(rb) {
		ra, rb = rb, ra
	}
	if len(rb)-len(ra) > 1 {
		return false
	}
	i := 0
	for i < len(ra) && ra[i] == rb[i] {
		i++
	}
	if len(ra) == len(rb) {
		return string(ra[min(i+1, len(ra)):]) == string(rb[min(i+1, len(rb)):])
	}
	return string(ra[i:]) == string(rb[i+1:])
}

func formsMatch(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	return len([]rune(a)) >= minFuzzySkeleton && len([]rune(b)) >= minFuzzySkeleton && editDistanceAtMostOne(a, b)
}

func fullMatch(a, b token) bool {
	for _, x := range a.forms {
		for _, y := range b.forms {
			if formsMatch(x, y) {
				return true
			}
		}
	}
	return false
}

// initialMatch pairs "J." with "Josiah". Latin vowels have no skeleton, so an
// abbreviated vowel ("A. Hassan") carries no evidence and simply passes.
func initialMatch(abbreviated, full token) bool {
	if abbreviated.forms != nil {
		return false
	}
	if abbreviated.initial == "" {
		return true
	}
	if full.forms == nil {
		return abbreviated.initial == full.initial
	}
	for _, form := range full.forms {
		if strings.HasPrefix(form, abbreviated.initial) {
			return true
		}
	}
	return false
}

// Match reports whether two names could be the same person. Every part of the
// shorter name must find a partner in the longer one, in any order, and at least
// two parts are needed: a lone given name fits too many people.
func Match(a, b string) bool {
	shorter, longer := tokenize(a), tokenize(b)
	if len(shorter) > len(longer) {
		shorter, longer = longer, shorter
	}
	if len(shorter) < 2 {
		return false
	}
	used := make([]bool, len(longer))
	strength, leftover := pairFull(shorter, longer, used)
	if strength < minStrongSkeleton {
		return false
	}
	return pairInitials(leftover, longer, used)
}

// pairFull claims one partner per token and returns the matched strength and the
// tokens still without a partner.
func pairFull(tokens, pool []token, used []bool) (int, []token) {
	strength := 0
	leftover := make([]token, 0, len(tokens))
	for _, t := range tokens {
		partner := -1
		for i, candidate := range pool {
			if !used[i] && fullMatch(t, candidate) {
				partner = i
				break
			}
		}
		if partner < 0 {
			leftover = append(leftover, t)
			continue
		}
		used[partner] = true
		strength += t.strength()
	}
	return strength, leftover
}

// pairInitials accepts an abbreviation on either side.
func pairInitials(tokens, pool []token, used []bool) bool {
	for _, t := range tokens {
		partner := -1
		for i, candidate := range pool {
			if !used[i] && (initialMatch(t, candidate) || initialMatch(candidate, t)) {
				partner = i
				break
			}
		}
		if partner < 0 {
			return false
		}
		used[partner] = true
	}
	return true
}

// MatchAny reports whether any name in as could be any name in bs.
func MatchAny(as, bs []string) bool {
	for _, a := range as {
		for _, b := range bs {
			if Match(a, b) {
				return true
			}
		}
	}
	return false
}
