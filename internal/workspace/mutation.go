// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package workspace

import (
	"strings"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/validation"
)

func SelectedProfile(doc *argument.Document, override string) validation.Profile {
	if override != "" {
		return validation.Profile(strings.ToLower(override))
	}
	if doc != nil {
		if profile, ok := doc.MetadataValue("profile"); ok {
			switch {
			case strings.EqualFold(profile, string(validation.ProfileCludia)), strings.EqualFold(profile, "workspace"):
				return validation.ProfileCludia
			default:
				return validation.Profile(strings.ToLower(profile))
			}
		}
	}
	return validation.ProfileConcludia
}

func canonicalizeLegacyProfile(doc *argument.Document) {
	if doc == nil {
		return
	}
	for index := range doc.Metadata {
		metadata := &doc.Metadata[index]
		if metadata.Key == "profile" && strings.EqualFold(metadata.Value, "workspace") {
			metadata.Value = string(validation.ProfileCludia)
			doc.LegacyWorkspaceProfile = true
		}
	}
}

func LoadValidated(path, override string) (*argument.Document, validation.Profile, []diagnostic.Diagnostic) {
	parsed := argfile.Load(path)
	profile := SelectedProfile(parsed.Document, override)
	if profile == validation.ProfileCludia {
		canonicalizeLegacyProfile(parsed.Document)
	}
	diagnostics := append([]diagnostic.Diagnostic(nil), parsed.Diagnostics...)
	if !diagnostic.HasErrors(diagnostics) {
		diagnostics = append(diagnostics, validation.Validate(parsed.Document, profile).Diagnostics...)
	}
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	return parsed.Document, profile, diagnostics
}

func ParseValidated(data []byte, profile validation.Profile) (*argument.Document, []diagnostic.Diagnostic) {
	parsed := argfile.Parse(string(data))
	if profile == validation.ProfileCludia {
		canonicalizeLegacyProfile(parsed.Document)
	}
	diagnostics := append([]diagnostic.Diagnostic(nil), parsed.Diagnostics...)
	if !diagnostic.HasErrors(diagnostics) {
		diagnostics = append(diagnostics, validation.Validate(parsed.Document, profile).Diagnostics...)
	}
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	return parsed.Document, diagnostics
}

// ValidateAndPersist validates the complete proposed document and, when
// persist is true, atomically saves it only after validation succeeds.
func ValidateAndPersist(path string, next *argument.Document, profile validation.Profile, persist bool) (validation.Result, error) {
	result := validation.Validate(next, profile)
	if !result.OK() || !persist {
		return result, nil
	}
	if err := argfile.SaveAtomic(path, next); err != nil {
		return result, err
	}
	return result, nil
}

// ValidateAndCreate validates a new workspace document and atomically creates
// it without overwriting an existing path.
func ValidateAndCreate(path string, next *argument.Document, profile validation.Profile) (validation.Result, error) {
	result := validation.Validate(next, profile)
	if !result.OK() {
		return result, nil
	}
	if err := argfile.CreateAtomic(path, next); err != nil {
		return result, err
	}
	return result, nil
}
