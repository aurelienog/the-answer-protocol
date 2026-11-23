package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

// Dispatcher routes incoming requests to registered commands and centrally
// logs command execution, responses, errors, and non-bus gameplay outcomes.
type Dispatcher struct {
	registry map[string]Command
	log      *slog.Logger
}

// NewDispatcher initializes an empty command registry.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		registry: make(map[string]Command),
	}
}

// SetLogger attaches a structured logger to the command dispatcher.
func (d *Dispatcher) SetLogger(log *slog.Logger) {
	if d != nil {
		d.log = log
	}
}

// Register binds a command implementation to its name.
func (d *Dispatcher) Register(cmd Command) {
	if cmd == nil {
		panic("cannot register a nil command")
	}

	name := strings.ToLower(strings.TrimSpace(cmd.Name()))
	if name == "" {
		panic("cannot register a command with an empty name")
	}

	d.registry[name] = cmd
}

// Dispatch parses a request, executes the corresponding command, and logs the outcome.
func (d *Dispatcher) Dispatch(
	ctx *Context,
	rawInput string,
) (protocol.Message, error) {
	request, err := protocol.ParseRequest(rawInput)
	if err != nil {
		return protocol.Message{}, err
	}
	if ctx == nil {
		return protocol.Message{}, fmt.Errorf("command context is nil")
	}

	cmdName := strings.ToLower(request.Command)

	if d.log != nil {
		d.log.Info("command.received",
			slog.String("player", ctx.PlayerName),
			slog.String("command", request.Command),
			slog.Any("args", request.Args),
		)
	}

	cmd, exists := d.registry[cmdName]
	if !exists {
		resp := protocol.ERR(protocol.CodeUnknownCommand, "UNKNOWN_COMMAND")
		d.logResponse(ctx.PlayerName, request.Command, resp)
		return resp, nil
	}

	ctx.RawArgs = request.Args
	ctx.RawText = request.RawArgs

	// Snapshot active quests before TALK so auto-completed quests can be audited centrally.
	var activeQuestsBefore map[string]world.PlayerQuestSummary
	if d.log != nil && cmdName == "talk" && ctx.World != nil {
		activeQuestsBefore = snapshotActiveQuests(ctx.World, ctx.PlayerName)
	}

	msg, execErr := cmd.Execute(ctx)

	if errors.Is(execErr, ErrQuit) {
		d.logResponse(ctx.PlayerName, request.Command, msg)
		return msg, execErr
	}

	if execErr != nil {
		if d.log != nil {
			d.log.Error("command.execution_error",
				slog.String("player", ctx.PlayerName),
				slog.String("command", request.Command),
				slog.String("error", execErr.Error()),
			)
		}
		return msg, execErr
	}

	d.logResponse(ctx.PlayerName, request.Command, msg)

	if d.log != nil && msg.Kind == protocol.KindOK {
		d.auditCommandOutcome(ctx, cmdName, msg.Payload, activeQuestsBefore)
	}

	return msg, nil
}

func (d *Dispatcher) logResponse(player, command string, msg protocol.Message) {
	if d.log == nil {
		return
	}

	switch msg.Kind {
	case protocol.KindOK:
		d.log.Info("command.response",
			slog.String("player", player),
			slog.String("command", command),
			slog.String("response_kind", "OK"),
		)
	case protocol.KindERR:
		attrs := []any{
			slog.String("player", player),
			slog.String("command", command),
			slog.String("response_kind", "ERR"),
			slog.Int("error_code", msg.Code),
			slog.String("error_name", msg.Name),
		}
		if msg.Code >= protocol.CodeInternalError {
			d.log.Error("command.response", attrs...)
		} else {
			d.log.Warn("command.response", attrs...)
		}
	}
}

// auditCommandOutcome logs state changes for commands that do not publish events
// on the internal event bus (TAKE, TALK, QUEST, GROUP CREATE).
func (d *Dispatcher) auditCommandOutcome(
	ctx *Context,
	cmdName string,
	payload string,
	activeQuestsBefore map[string]world.PlayerQuestSummary,
) {
	switch cmdName {
	case "take":
		itemID := strings.TrimPrefix(payload, "taken=")
		roomID := ""
		if ctx.World != nil {
			roomID, _ = ctx.World.GetPlayerRoom(ctx.PlayerName)
		}
		d.log.Info("world.item_taken",
			slog.String("player", ctx.PlayerName),
			slog.String("item_id", itemID),
			slog.String("room_id", roomID),
		)

	case "quest":
		var q world.QuestSummary
		if err := json.Unmarshal([]byte(payload), &q); err == nil {
			d.log.Info("quest.accepted",
				slog.String("player", ctx.PlayerName),
				slog.String("quest_id", q.ID),
				slog.String("quest_name", q.Name),
			)
		}

	case "talk":
		var conv world.NPCConversation
		if err := json.Unmarshal([]byte(payload), &conv); err == nil {
			attrs := []any{
				slog.String("player", ctx.PlayerName),
				slog.String("npc_id", conv.NPC.ID),
				slog.String("npc_name", conv.NPC.Name),
			}
			if conv.Quest != nil {
				attrs = append(attrs,
					slog.String("quest_id", conv.Quest.ID),
					slog.String("quest_status", string(conv.QuestStatus)),
				)
				if conv.Quest.Progress != "" {
					attrs = append(attrs, slog.String("quest_progress", conv.Quest.Progress))
				}
			}
			d.log.Info("world.npc_interaction", attrs...)

			if len(activeQuestsBefore) > 0 && ctx.World != nil {
				activeAfter := snapshotActiveQuests(ctx.World, ctx.PlayerName)
				for questID, q := range activeQuestsBefore {
					if _, stillActive := activeAfter[questID]; !stillActive {
						d.log.Info("quest.completed",
							slog.String("player", ctx.PlayerName),
							slog.String("quest_id", q.ID),
							slog.String("quest_name", q.Name),
							slog.String("npc_id", conv.NPC.ID),
						)
					}
				}
			}
		}

	case "group":
		if len(ctx.RawArgs) > 0 && strings.EqualFold(ctx.RawArgs[0], "CREATE") {
			d.log.Info("group.created",
				slog.String("player", ctx.PlayerName),
				slog.String("details", payload),
			)
		}
	}
}

func snapshotActiveQuests(gameWorld *world.World, playerName string) map[string]world.PlayerQuestSummary {
	quests, err := gameWorld.GetPlayerQuests(playerName)
	if err != nil {
		return nil
	}
	active := make(map[string]world.PlayerQuestSummary)
	for _, q := range quests {
		if q.Status == world.QuestStatusActive {
			active[q.ID] = q
		}
	}
	return active
}