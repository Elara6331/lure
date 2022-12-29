package cliutils

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/pager"
)

// YesNoPrompt asks the user a yes or no question, using def as the default answer
func YesNoPrompt(msg string, def bool) (bool, error) {
	var answer bool
	err := survey.AskOne(
		&survey.Confirm{
			Message: msg,
			Default: def,
		},
		&answer,
	)
	return answer, err
}

// PromptViewScript asks the user if they'd like to see a script,
// shows it if they answer yes, then asks if they'd still like to
// continue, and exits if they answer no.
func PromptViewScript(script, name, style string) error {
	view, err := YesNoPrompt("Would you like to view the build script for "+name, false)
	if err != nil {
		return err
	}

	if view {
		err = ShowScript(script, name, style)
		if err != nil {
			return err
		}

		cont, err := YesNoPrompt("Would you still like to continue?", false)
		if err != nil {
			return err
		}

		if !cont {
			log.Fatal("User chose not to continue after reading script").Send()
		}
	}

	return nil
}

// ShowScript uses the built-in pager to display a script at a
// given path, in the given syntax highlighting style.
func ShowScript(path, name, style string) error {
	scriptFl, err := os.Open(path)
	if err != nil {
		return err
	}
	defer scriptFl.Close()

	str, err := pager.SyntaxHighlightBash(scriptFl, style)
	if err != nil {
		return err
	}

	pgr := pager.New(name, str)
	return pgr.Run()
}

// FlattenPkgs attempts to flatten the a map of slices of packages into a single slice
// of packages by prompting the user if multiple packages match.
func FlattenPkgs(found map[string][]db.Package, verb string) []db.Package {
	var outPkgs []db.Package
	for _, pkgs := range found {
		if len(pkgs) > 1 {
			choices, err := PkgPrompt(pkgs, verb)
			if err != nil {
				log.Fatal("Error prompting for choice of package").Send()
			}
			outPkgs = append(outPkgs, choices...)
		} else if len(pkgs) == 1 {
			outPkgs = append(outPkgs, pkgs[0])
		}
	}
	return outPkgs
}

// PkgPrompt asks the user to choose between multiple packages.
// The user may choose multiple packages.
func PkgPrompt(options []db.Package, verb string) ([]db.Package, error) {
	names := make([]string, len(options))
	for i, option := range options {
		names[i] = option.Repository + "/" + option.Name + " " + option.Version
	}

	prompt := &survey.MultiSelect{
		Options: names,
		Message: "Choose which package(s) to " + verb,
	}

	var choices []int
	err := survey.AskOne(prompt, &choices)
	if err != nil {
		return nil, err
	}

	out := make([]db.Package, len(choices))
	for i, choiceIndex := range choices {
		out[i] = options[choiceIndex]
	}

	return out, nil
}
