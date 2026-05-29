package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"raven/internal/app"
	"raven/internal/domain"
	"raven/internal/storage"
)

func Run(args []string, configDir string, stdout, stderr io.Writer) error {
	return RunWithInput(args, configDir, os.Stdin, stdout, stderr)
}

func RunWithInput(args []string, configDir string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return nil
	}

	switch args[0] {
	case "ci":
		return runCI(args[1:], configDir, stdout, stderr)
	case "event":
		return runEvent(args[1:], configDir, stdin, stdout, stderr)
	case "timeline":
		return runTimeline(args[1:], configDir, stdout, stderr)
	default:
		err := fmt.Errorf("unknown command %q", args[0])
		fmt.Fprintln(stderr, err)
		return err
	}
}

func runCI(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		err := errors.New("ci subcommand is required")
		fmt.Fprintln(stderr, err)
		return err
	}

	switch args[0] {
	case "add":
		return runCIAdd(args[1:], configDir, stdout, stderr)
	case "list":
		return runCIList(args[1:], configDir, stdout, stderr)
	case "show":
		return runCIShow(args[1:], configDir, stdout, stderr)
	default:
		err := fmt.Errorf("unknown ci subcommand %q", args[0])
		fmt.Fprintln(stderr, err)
		return err
	}
}

func runCIAdd(args []string, configDir string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("ci add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	ciID := flags.String("ci-id", "", "CI ID")
	category := flags.String("category", "", "component category")
	manufacturer := flags.String("manufacturer", "", "component manufacturer")
	model := flags.String("model", "", "component model")
	serialNumber := flags.String("serial-number", "", "component serial number")
	notes := flags.String("notes", "", "component notes")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("ci add does not accept positional arguments: %s", strings.Join(flags.Args(), " "))
		fmt.Fprintln(stderr, err)
		return err
	}

	component := domain.Component{
		CIID:         *ciID,
		Category:     domain.ComponentCategory(*category),
		Manufacturer: *manufacturer,
		Model:        *model,
		SerialNumber: *serialNumber,
		Notes:        *notes,
	}
	components, inventory, err := loadInventory(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if err := inventory.Add(component); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	components = inventory.List()
	if err := storage.SaveComponents(app.ComponentsPath(configDir), components); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	fmt.Fprintf(stdout, "added CI %s\n", strings.TrimSpace(component.CIID))
	return nil
}

func runCIList(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) != 0 {
		err := errors.New("ci list does not accept arguments")
		fmt.Fprintln(stderr, err)
		return err
	}

	components, _, err := loadInventory(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if len(components) == 0 {
		fmt.Fprintln(stdout, "No CIs yet.")
		return nil
	}

	fmt.Fprintln(stdout, "CI ID\tCategory\tName")
	for _, component := range components {
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", component.CIID, component.Category, component.DisplayName())
	}
	return nil
}

func runCIShow(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) != 1 {
		err := errors.New("ci show requires CI ID")
		fmt.Fprintln(stderr, err)
		return err
	}

	_, inventory, err := loadInventory(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	component, err := inventory.Get(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	fmt.Fprintf(stdout, "CI ID: %s\n", component.CIID)
	fmt.Fprintf(stdout, "Category: %s\n", component.Category)
	if strings.TrimSpace(component.Manufacturer) != "" {
		fmt.Fprintf(stdout, "Manufacturer: %s\n", component.Manufacturer)
	}
	fmt.Fprintf(stdout, "Model: %s\n", component.Model)
	if strings.TrimSpace(component.SerialNumber) != "" {
		fmt.Fprintf(stdout, "Serial Number: %s\n", component.SerialNumber)
	}
	if strings.TrimSpace(component.Notes) != "" {
		fmt.Fprintf(stdout, "Notes: %s\n", component.Notes)
	}
	return nil
}

func runEvent(args []string, configDir string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		err := errors.New("event subcommand is required")
		fmt.Fprintln(stderr, err)
		return err
	}

	switch args[0] {
	case "add":
		return runEventAdd(args[1:], configDir, stdout, stderr)
	case "capture":
		return runEventCapture(args[1:], configDir, stdout, stderr)
	case "ingest":
		return runEventIngest(args[1:], configDir, stdin, stdout, stderr)
	default:
		err := fmt.Errorf("unknown event subcommand %q", args[0])
		fmt.Fprintln(stderr, err)
		return err
	}
}

func runEventAdd(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		err := errors.New("event add requires CI ID")
		fmt.Fprintln(stderr, err)
		return err
	}
	ciID := args[0]

	flags := flag.NewFlagSet("event add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	eventType := flags.String("type", "", "event type")
	severity := flags.String("severity", "", "event severity")
	status := flags.String("status", "open", "event status")
	summary := flags.String("summary", "", "event summary")
	details := flags.String("details", "", "event details")
	source := flags.String("source", "", "event source")
	externalID := flags.String("external-id", "", "source event ID")
	raw := flags.String("raw", "", "raw source evidence")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("event add does not accept positional arguments: %s", strings.Join(flags.Args(), " "))
		fmt.Fprintln(stderr, err)
		return err
	}

	_, inventory, err := loadInventory(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if _, err := inventory.Get(ciID); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	now := time.Now().UTC()
	eventID := fmt.Sprintf("evt-%d", now.UnixNano())
	dedupKey := buildDedupKey(*source, *externalID, eventID)
	event := domain.Event{
		ID:         eventID,
		CIID:       ciID,
		Type:       *eventType,
		Severity:   *severity,
		Status:     *status,
		Summary:    *summary,
		Details:    *details,
		Source:     *source,
		ExternalID: *externalID,
		DedupKey:   dedupKey,
		ObservedAt: now,
		IngestedAt: now,
		Raw:        *raw,
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	events = append(events, event)
	if err := storage.SaveEvents(app.EventsPath(configDir), events); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	fmt.Fprintf(stdout, "added event %s for CI %s\n", eventID, strings.TrimSpace(ciID))
	return nil
}

func runEventCapture(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		err := errors.New("event capture requires CI ID")
		fmt.Fprintln(stderr, err)
		return err
	}
	ciID := args[0]

	flags := flag.NewFlagSet("event capture", flag.ContinueOnError)
	flags.SetOutput(stderr)
	eventType := flags.String("type", "observation", "event type")
	severity := flags.String("severity", "info", "event severity")
	status := flags.String("status", "open", "event status")
	summary := flags.String("summary", "", "event summary")
	text := flags.String("text", "", "captured event text")
	source := flags.String("source", "", "event source")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("event capture does not accept positional arguments: %s", strings.Join(flags.Args(), " "))
		fmt.Fprintln(stderr, err)
		return err
	}
	if strings.TrimSpace(*text) == "" {
		err := errors.New("event capture requires --text")
		fmt.Fprintln(stderr, err)
		return err
	}

	_, inventory, err := loadInventory(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if _, err := inventory.Get(ciID); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	now := time.Now().UTC()
	eventID := fmt.Sprintf("evt-%d", now.UnixNano())
	eventSummary := strings.TrimSpace(*summary)
	if eventSummary == "" {
		eventSummary = firstLine(*text)
	}
	event := domain.Event{
		ID:         eventID,
		CIID:       ciID,
		Type:       *eventType,
		Severity:   *severity,
		Status:     *status,
		Summary:    eventSummary,
		Details:    *text,
		Source:     *source,
		DedupKey:   buildDedupKey(*source, "", eventID),
		ObservedAt: now,
		IngestedAt: now,
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	events = append(events, event)
	if err := storage.SaveEvents(app.EventsPath(configDir), events); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	fmt.Fprintf(stdout, "captured event %s for CI %s\n", eventID, strings.TrimSpace(ciID))
	return nil
}

func runEventIngest(args []string, configDir string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("event ingest", flag.ContinueOnError)
	flags.SetOutput(stderr)
	source := flags.String("source", "", "event source")
	file := flags.String("file", "", "normalized event JSON file")
	useStdin := flags.Bool("stdin", false, "read normalized event JSON from stdin")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("event ingest does not accept positional arguments: %s", strings.Join(flags.Args(), " "))
		fmt.Fprintln(stderr, err)
		return err
	}

	filePath := strings.TrimSpace(*file)
	if (filePath == "" && !*useStdin) || (filePath != "" && *useStdin) {
		err := errors.New("event ingest requires exactly one of --file or --stdin")
		fmt.Fprintln(stderr, err)
		return err
	}

	var data []byte
	var err error
	if *useStdin {
		if stdin == nil {
			err := errors.New("event ingest stdin is unavailable")
			fmt.Fprintln(stderr, err)
			return err
		}
		data, err = io.ReadAll(stdin)
		if err != nil {
			err = fmt.Errorf("read ingest stdin: %w", err)
			fmt.Fprintln(stderr, err)
			return err
		}
	} else {
		data, err = os.ReadFile(filePath)
		if err != nil {
			err = fmt.Errorf("read ingest file: %w", err)
			fmt.Fprintln(stderr, err)
			return err
		}
	}

	var event domain.Event
	if err := json.Unmarshal(data, &event); err != nil {
		err = fmt.Errorf("decode ingest event: %w", err)
		fmt.Fprintln(stderr, err)
		return err
	}
	event = event.Normalize()
	if strings.TrimSpace(*source) != "" {
		event.Source = strings.TrimSpace(*source)
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d", time.Now().UTC().UnixNano())
	}
	if event.Status == "" {
		event.Status = "open"
	}
	if event.ExternalID == "" && event.DedupKey == "" {
		err := errors.New("event ingest requires external_id or dedup_key")
		fmt.Fprintln(stderr, err)
		return err
	}
	if event.ExternalID != "" {
		event.DedupKey = buildDedupKey(event.Source, event.ExternalID, event.ID)
	}
	if event.IngestedAt.IsZero() {
		event.IngestedAt = time.Now().UTC()
	}

	_, inventory, err := loadInventory(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if _, err := inventory.Get(event.CIID); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	events = append(events, event)
	if err := storage.SaveEvents(app.EventsPath(configDir), events); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	fmt.Fprintf(stdout, "ingested event %s for CI %s\n", event.ID, event.CIID)
	return nil
}

func runTimeline(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) != 1 {
		err := errors.New("timeline requires CI ID")
		fmt.Fprintln(stderr, err)
		return err
	}

	_, inventory, err := loadInventory(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	ciID := strings.TrimSpace(args[0])
	if _, err := inventory.Get(ciID); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	fmt.Fprintf(stdout, "Timeline for %s\n", ciID)
	matched := 0
	for _, event := range events {
		if event.CIID != ciID {
			continue
		}
		matched++
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", event.ObservedAt.Format(time.RFC3339), event.Type, event.Severity, event.Summary)
	}
	if matched == 0 {
		fmt.Fprintln(stdout, "No events yet.")
	}
	return nil
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		return strings.TrimSpace(text[:index])
	}
	return text
}

func buildDedupKey(source, externalID, eventID string) string {
	source = strings.TrimSpace(source)
	externalID = strings.TrimSpace(externalID)
	if externalID != "" {
		return source + ":" + externalID
	}
	return source + ":" + eventID
}

func loadInventory(configDir string) ([]domain.Component, *domain.Inventory, error) {
	components, err := storage.LoadComponents(app.ComponentsPath(configDir))
	if err != nil {
		return nil, nil, err
	}

	inventory := domain.NewInventory()
	for _, component := range components {
		if err := inventory.Add(component); err != nil {
			return nil, nil, err
		}
	}
	return components, inventory, nil
}
