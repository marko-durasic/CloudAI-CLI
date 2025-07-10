package llm

import (
    "regexp"
    "strconv"
    "strings"
)

// DataProtector redacts sensitive AWS identifiers from prompts before they
// are sent to external LLM providers and allows them to be restored afterwards.
//
// The strategy is simple but effective:
// 1. Identify well-known sensitive patterns via regular expressions
//    (ARNs, account IDs, IP addresses, S3 URLs, etc.).
// 2. Replace each match with a deterministic placeholder such as
//    [[ARN_1]], [[ACCOUNT_ID_2]], etc.
// 3. Keep an in-memory map so the original values can be re-hydrated once the
//    LLM has produced its answer.
//
// NOTE: The mapping is intentionally NOT written to disk to avoid persisting
// sensitive material. Callers that need cross-process re-hydration can marshal
// the map to an encrypted store (e.g. DynamoDB + KMS) – left to the caller.
//
// The protector is *stateless* from the point of view of concurrent requests;
// create a fresh instance per request or guard access with a mutex if you want
// to reuse one instance.

type DataProtector struct {
    // placeholder -> original
    replacements map[string]string
    nextIndex    int
}

func NewDataProtector() *DataProtector {
    return &DataProtector{
        replacements: make(map[string]string),
        nextIndex:    1,
    }
}

// Scrub replaces sensitive tokens in the given text with placeholders and
// returns the scrubbed text. The mapping is stored internally so the caller can
// later reverse the process via Unscrub.
func (p *DataProtector) Scrub(text string) string {
    if text == "" {
        return text
    }

    // Ordered list matters – longer / more specific patterns first.
    patterns := []struct {
        name string
        re   *regexp.Regexp
    }{
        {"ARN", regexp.MustCompile(`arn:[A-Za-z0-9\-_:/.]+`)},
        {"ACCOUNT_ID", regexp.MustCompile(`\b\d{12}\b`)},
        {"IP", regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}\b`)},
        {"S3", regexp.MustCompile(`s3://[A-Za-z0-9.\-_/]+`)},
    }

    scrubbed := text
    for _, pat := range patterns {
        scrubbed = p.replaceAll(scrubbed, pat.re, pat.name)
    }

    return scrubbed
}

func (p *DataProtector) replaceAll(input string, re *regexp.Regexp, tag string) string {
    matches := re.FindAllStringIndex(input, -1)
    if len(matches) == 0 {
        return input
    }

    // Work with a builder to avoid repeated string allocations.
    var b strings.Builder
    last := 0
    for _, loc := range matches {
        start, end := loc[0], loc[1]
        sensitive := input[start:end]

        placeholder := p.buildPlaceholder(tag)
        p.replacements[placeholder] = sensitive

        b.WriteString(input[last:start])
        b.WriteString(placeholder)
        last = end
    }
    b.WriteString(input[last:])
    return b.String()
}

func (p *DataProtector) buildPlaceholder(tag string) string {
    ph := "[[" + tag + "_" + strconv.Itoa(p.nextIndex) + "]]"
    p.nextIndex++
    return ph
}

// Unscrub reverses placeholders previously inserted by Scrub.
// If the text does not contain any placeholders known to this protector the
// original text is returned unchanged.
func (p *DataProtector) Unscrub(text string) string {
    if len(p.replacements) == 0 || text == "" {
        return text
    }

    result := text
    for placeholder, original := range p.replacements {
        result = strings.ReplaceAll(result, placeholder, original)
    }
    return result
}