package main

import "github.com/jason0x43/go-alfred"

type RefreshCommand struct{}

func (t RefreshCommand) Keyword() string {
	return "refresh"
}

func (t RefreshCommand) IsEnabled() bool {
	return isAuthorized()
}

func (t RefreshCommand) MenuItem() alfred.Item {
	return alfred.Item{
		Title:        t.Keyword(),
		Autocomplete: t.Keyword(),
		SubtitleAll:  "Refresh status from Nest.com",
		Valid:        alfred.Invalid,
	}
}

func (t RefreshCommand) Items(sync, query string) ([]alfred.Item, error) {
	if err := refresh(); err != nil {
		return []alfred.Item{}, err
	}

	return []alfred.Item{alfred.Item{
		Title: "All freshened up!",
		Valid: alfred.Invalid,
	}}, nil
}
