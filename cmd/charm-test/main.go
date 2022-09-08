package main

import (
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/commands/charm/input"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/commands/charm/text"
	"github.com/digitalocean/doctl/commands/charm/textbox"
)

func main() {
	var err error
	choice, err := confirm.New("wanna see a magic trick?",
		confirm.WithDefaultChoice(confirm.Yes),
		confirm.WithPersistPrompt(confirm.PersistPromptIfNo),
	).Prompt()
	if err != nil {
		fmt.Println(err)
	}
	if choice == confirm.Yes {
		fmt.Println("👻")
	}
	i := input.New("app name:", input.WithRequired())
	_, err = i.Prompt()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(
		text.Checkmark,
		text.Checkmark.Inherit(text.Success),
	)

	fmt.Println(
		text.Success.WithString("woo!"), text.Success.S("woo 2!"),
	)

	if err := template.PrintE(heredoc.Doc(`
		--- template ---
		This is an example template.

		Another line.

		{{ success "maybe some success output" }}
		{{ success checkmark }} just the checkmark.
		{{ success (join " " (checkmark) "good job!") }}
		{{ error (join " " (checkmark) "we're both confused.") }}
		{{ warning (print promptPrefix " try again?") }}
		{{ error (join " " (crossmark) "there we go.") }}

		{{ success (bold "full send let's go!!!!") }}
		{{ bold (success "full send let's go!!!!") }}
		
		{{ bold (underline "underline behaves very strangely") }}
		{{ underline (bold "underline behaves very strangely") }}
		
		{{ success (underline "underline behaves very strangely") }}
		{{ underline (success "underline behaves very strangely") }}

		{{ newTextBox.Success.S "i'm in a box!" }}
	`), nil); err != nil {
		panic(err)
	}

	img := "yeet/yote:dev"
	dur := 23*time.Minute + 37*time.Second
	fmt.Fprintf(
		textbox.New().Success(),
		"%s Successfully built %s in %s",
		text.Checkmark.Success(),
		text.Success.S(img),
		text.Warning.S(dur.Truncate(time.Second).String()),
	)

	if err := template.BufferedE(textbox.New().Success(), heredoc.Doc(`
		{{ success checkmark }} Successfully built {{ success  .img }} in {{ warning (duration .dur) }}`,
	), map[string]any{
		"img": img,
		"dur": dur,
	}); err != nil {
		panic(err)
	}

	template.Buffered(textbox.New().Success(), heredoc.Doc(`
		{{ success checkmark }} Successfully built {{ success  .img }} in {{ warning (duration .dur) }}`,
	), map[string]any{
		"img": img,
		"dur": dur,
	})
}
