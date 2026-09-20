package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"raven/internal/app"
	"raven/internal/domain"
	ravenmcp "raven/internal/mcp"
	"raven/internal/nextgenmcp"
	"raven/internal/service"
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
	case "alias":
		return runAlias(args[1:], configDir, stdout, stderr)
	case "event":
		return runEvent(args[1:], configDir, stdin, stdout, stderr)
	case "timeline":
		return runTimeline(args[1:], configDir, stdout, stderr)
	case "metadata":
		return runMetadata(args[1:], configDir, stdout, stderr)
	case "mcp":
		return ravenmcp.ServeStdio(args[1:], configDir, stderr)
	case "nextgen-mcp":
		return nextgenmcp.ServeStdio(args[1:], stderr)
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
	flags := flag.NewFlagSet("ci list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	category := flags.String("category", "", "exact-match filter on component category")
	prefix := flags.String("prefix", "", "case-sensitive prefix filter on ci_id")
	query := flags.String("query", "", "case-insensitive substring search across ci_id, model, and notes")
	limit := flags.Int("limit", 0, "maximum number of CIs to return after filtering (0 or negative means no cap)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := errors.New("ci list does not accept positional arguments")
		fmt.Fprintln(stderr, err)
		return err
	}

	all, _, err := loadInventory(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	filter := service.ListFilter{
		Category: strings.TrimSpace(*category),
		Prefix:   *prefix,
		Query:    strings.TrimSpace(*query),
		Limit:    *limit,
	}
	components, err := service.New(configDir).ListCIsWithFilter(filter)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	if len(components) == 0 {
		if len(all) == 0 {
			fmt.Fprintln(stdout, "No CIs yet.")
		} else {
			fmt.Fprintln(stdout, "No CIs matched the provided filters.")
		}
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

func runAlias(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		err := errors.New("alias subcommand is required")
		fmt.Fprintln(stderr, err)
		return err
	}

	switch args[0] {
	case "add":
		return runAliasAdd(args[1:], configDir, stdout, stderr)
	case "list":
		return runAliasList(args[1:], configDir, stdout, stderr)
	case "resolve":
		return runAliasResolve(args[1:], configDir, stdout, stderr)
	default:
		err := fmt.Errorf("unknown alias subcommand %q", args[0])
		fmt.Fprintln(stderr, err)
		return err
	}
}

func runAliasAdd(args []string, configDir string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("alias add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	ciID := flags.String("ci-id", "", "canonical CI ID")
	source := flags.String("source", "", "alias source namespace")
	aliasType := flags.String("type", "", "alias type")
	value := flags.String("value", "", "alias value")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("alias add does not accept positional arguments: %s", strings.Join(flags.Args(), " "))
		fmt.Fprintln(stderr, err)
		return err
	}

	alias := domain.Alias{
		CIID:   *ciID,
		Source: *source,
		Type:   domain.AliasType(*aliasType),
		Value:  *value,
	}
	if err := alias.Validate(); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	_, inventory, err := loadInventory(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if _, err := inventory.Get(alias.CIID); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	_, registry, err := loadAliasRegistry(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if err := registry.Add(alias); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	aliases := registry.List()
	if err := storage.SaveAliases(app.AliasesPath(configDir), aliases); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	alias = alias.Normalize()
	fmt.Fprintf(stdout, "added alias %s %s %s -> %s\n", alias.Source, alias.Type, alias.Value, alias.CIID)
	return nil
}

func runAliasList(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) != 0 {
		err := errors.New("alias list does not accept arguments")
		fmt.Fprintln(stderr, err)
		return err
	}

	aliases, _, err := loadAliasRegistry(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if len(aliases) == 0 {
		fmt.Fprintln(stdout, "No aliases yet.")
		return nil
	}

	fmt.Fprintln(stdout, "Source\tType\tValue\tCI ID")
	for _, alias := range aliases {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", alias.Source, alias.Type, alias.Value, alias.CIID)
	}
	return nil
}

func runAliasResolve(args []string, configDir string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("alias resolve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	source := flags.String("source", "", "alias source namespace")
	aliasType := flags.String("type", "", "alias type")
	value := flags.String("value", "", "alias value")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("alias resolve does not accept positional arguments: %s", strings.Join(flags.Args(), " "))
		fmt.Fprintln(stderr, err)
		return err
	}

	key := domain.AliasKey{Source: *source, Type: domain.AliasType(*aliasType), Value: *value}
	if err := key.Validate(); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	_, registry, err := loadAliasRegistry(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	ciID, err := registry.Resolve(key)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	fmt.Fprintln(stdout, ciID)
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
	dedupKey := service.BuildDedupKey(*source, *externalID, eventID)
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
		DedupKey:   service.BuildDedupKey(*source, "", eventID),
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

	var payload service.RecordEventInput
	if err := json.Unmarshal(data, &payload); err != nil {
		err = fmt.Errorf("decode ingest event: %w", err)
		fmt.Fprintln(stderr, err)
		return err
	}
	payload.SourceOverride = *source
	event, err := service.New(configDir).RecordEvent(payload)
	if err != nil {
		if errors.Is(err, service.ErrMissingEventDedup) {
			err = errors.New("event ingest requires external_id or dedup_key")
		}
		if errors.Is(err, service.ErrMissingEventIdentity) {
			err = errors.New("event ingest requires ci_id or ci_ref")
		}
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

func runMetadata(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		err := errors.New("metadata subcommand is required")
		fmt.Fprintln(stderr, err)
		return err
	}

	switch args[0] {
	case "add":
		return runMetadataAdd(args[1:], configDir, stdout, stderr)
	case "list":
		return runMetadataList(args[1:], configDir, stdout, stderr)
	case "show":
		return runMetadataShow(args[1:], configDir, stdout, stderr)
	default:
		err := fmt.Errorf("unknown metadata subcommand %q", args[0])
		fmt.Fprintln(stderr, err)
		return err
	}
}

func runMetadataAdd(args []string, configDir string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("metadata add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	ciID := flags.String("ci-id", "", "CI ID this metadata entry belongs to")
	attribute := flags.String("attribute", "", "comma-separated key=value pairs (k1=v1,k2=v2). Values are auto-typed: 'true'/'false' -> bool, parseable as float -> number, else string.")
	relationship := flags.String("relationship", "", "comma-separated target_ci_id=kind pairs (t1=k1,t2=k2)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("metadata add does not accept positional arguments: %s", strings.Join(flags.Args(), " "))
		fmt.Fprintln(stderr, err)
		return err
	}
	if strings.TrimSpace(*ciID) == "" {
		err := errors.New("metadata add requires --ci-id")
		fmt.Fprintln(stderr, err)
		return err
	}

	sidecar, err := loadMetadata(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	// Find existing entry for ci-id; if absent, append a new one.
	idx := -1
	for i, entry := range sidecar.Entries {
		if strings.TrimSpace(entry.CIID) == strings.TrimSpace(*ciID) {
			idx = i
			break
		}
	}
	var entry domain.CIMetadataEntry
	if idx >= 0 {
		entry = sidecar.Entries[idx]
	} else {
		entry = domain.CIMetadataEntry{CIID: *ciID}
	}

	if entry.Attributes == nil {
		entry.Attributes = map[string]domain.TypedValue{}
	}

	// Parse --attribute pairs
	if *attribute != "" {
		for _, pair := range strings.Split(*attribute, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			eq := strings.IndexByte(pair, '=')
			if eq <= 0 || eq == len(pair)-1 {
				err := fmt.Errorf("invalid --attribute entry %q: expected key=value", pair)
				fmt.Fprintln(stderr, err)
				return err
			}
			key := strings.TrimSpace(pair[:eq])
			rawValue := pair[eq+1:]
			if key == "" {
				err := errors.New("--attribute has empty key")
				fmt.Fprintln(stderr, err)
				return err
			}
			entry.Attributes[key] = parseTypedValue(rawValue)
		}
	}

	// Parse --relationship pairs
	if *relationship != "" {
		for _, pair := range strings.Split(*relationship, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			eq := strings.IndexByte(pair, '=')
			if eq <= 0 || eq == len(pair)-1 {
				err := fmt.Errorf("invalid --relationship entry %q: expected target_ci_id=kind", pair)
				fmt.Fprintln(stderr, err)
				return err
			}
			target := strings.TrimSpace(pair[:eq])
			kind := strings.TrimSpace(pair[eq+1:])
			if target == "" || kind == "" {
				err := fmt.Errorf("--relationship pair %q has empty target or kind", pair)
				fmt.Fprintln(stderr, err)
				return err
			}
			entry.Relationships = append(entry.Relationships, domain.CIRelationship{
				TargetCIID: target,
				Kind:       kind,
			})
		}
	}

	// Upsert: replace or append the entry.
	if idx >= 0 {
		sidecar.Entries[idx] = entry
	} else {
		sidecar.Entries = append(sidecar.Entries, entry)
	}

	if err := storage.SaveMetadata(app.MetadataPath(configDir), sidecar); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	fmt.Fprintf(stdout, "updated metadata for %s\n", strings.TrimSpace(*ciID))
	return nil
}

// parseTypedValue auto-detects the TypedValue shape from a raw string value.
// 'true'/'false' (case-insensitive) -> BoolValue; parseable as float64 ->
// NumberValue; otherwise -> StringValue.
func parseTypedValue(raw string) domain.TypedValue {
	raw = strings.TrimSpace(raw)
	switch strings.ToLower(raw) {
	case "true":
		return domain.BoolValue(true)
	case "false":
		return domain.BoolValue(false)
	}
	if n, err := strconv.ParseFloat(raw, 64); err == nil {
		return domain.NumberValue(n)
	}
	return domain.StringValue(raw)
}

func runMetadataList(args []string, configDir string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("metadata list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	limit := flags.Int("limit", 0, "maximum number of entries to print (0 or negative means no cap)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := errors.New("metadata list does not accept positional arguments")
		fmt.Fprintln(stderr, err)
		return err
	}

	sidecar, err := loadMetadata(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	if len(sidecar.Entries) == 0 {
		fmt.Fprintln(stdout, "No metadata yet.")
		return nil
	}

	fmt.Fprintln(stdout, "CI ID\tAttributes\tRelationships")
	printed := 0
	for _, entry := range sidecar.Entries {
		if *limit > 0 && printed >= *limit {
			break
		}
		fmt.Fprintf(stdout, "%s\t%d\t%d\n",
			strings.TrimSpace(entry.CIID),
			len(entry.Attributes),
			len(entry.Relationships),
		)
		printed++
	}
	return nil
}

func runMetadataShow(args []string, configDir string, stdout, stderr io.Writer) error {
	if len(args) != 1 {
		err := errors.New("metadata show requires exactly one positional argument: <ci-id>")
		fmt.Fprintln(stderr, err)
		return err
	}
	ciID := strings.TrimSpace(args[0])

	sidecar, err := loadMetadata(configDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	var found *domain.CIMetadataEntry
	for i := range sidecar.Entries {
		if strings.TrimSpace(sidecar.Entries[i].CIID) == ciID {
			found = &sidecar.Entries[i]
			break
		}
	}
	if found == nil {
		err := fmt.Errorf("no metadata found for CI %q", ciID)
		fmt.Fprintln(stderr, err)
		return err
	}

	fmt.Fprintf(stdout, "CI ID: %s\n", ciID)

	if len(found.Attributes) == 0 && len(found.Relationships) == 0 {
		fmt.Fprintln(stdout, "  (no attributes or relationships)")
		return nil
	}

	if len(found.Attributes) > 0 {
		fmt.Fprintln(stdout, "Attributes:")
		// Stable iteration order via sort.Strings on keys.
		keys := make([]string, 0, len(found.Attributes))
		for k := range found.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(stdout, "  %s = %s\n", k, formatTypedValue(found.Attributes[k]))
		}
	}

	if len(found.Relationships) > 0 {
		fmt.Fprintln(stdout, "Relationships:")
		for _, rel := range found.Relationships {
			fmt.Fprintf(stdout, "  %s -> %s\n", rel.TargetCIID, rel.Kind)
		}
	}
	return nil
}

// formatTypedValue renders a TypedValue as a string for CLI output.
func formatTypedValue(v domain.TypedValue) string {
	switch raw := v.Raw().(type) {
	case bool:
		return strconv.FormatBool(raw)
	case float64:
		return strconv.FormatFloat(raw, 'g', -1, 64)
	case string:
		return raw
	default:
		return ""
	}
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		return strings.TrimSpace(text[:index])
	}
	return text
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

func loadAliasRegistry(configDir string) ([]domain.Alias, *domain.AliasRegistry, error) {
	aliases, err := storage.LoadAliases(app.AliasesPath(configDir))
	if err != nil {
		return nil, nil, err
	}

	registry := domain.NewAliasRegistry()
	for _, alias := range aliases {
		if err := registry.Add(alias); err != nil {
			return nil, nil, err
		}
	}
	return aliases, registry, nil
}

func loadMetadata(configDir string) (domain.MetadataSidecar, error) {
	sidecar, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		return domain.MetadataSidecar{}, err
	}
	return sidecar, nil
}
