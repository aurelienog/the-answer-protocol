package commands

import (
    "strings"
    "unicode"

    "the-answer-protocol/internal/protocol"
)

type ChatCommand struct{}

func NewChatCommand() *ChatCommand {
    return &ChatCommand{}
}

func (c *ChatCommand) Name() string {
    return "CHAT"
}

func (c *ChatCommand) Execute(ctx *Context) (protocol.Message, error) {
    if ctx == nil || ctx.World == nil {
        return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
    }

    if len(ctx.RawArgs) < 2 {
        return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
    }

    scope := strings.ToUpper(ctx.RawArgs[0])

    // RawText preserves the original spacing of the message.
    separator := strings.IndexFunc(ctx.RawText, unicode.IsSpace)
    if separator == -1 {
        return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
    }

    message := strings.TrimSpace(ctx.RawText[separator:])

    if message == "" {
        return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
    }

    event, err := ctx.World.SendChat(ctx.PlayerName, scope, message)
    if err != nil {
        return mapGroupError(err), nil
    }

    if ctx.Events != nil {
        ctx.Events.Publish(event)
    }

    return protocol.OK(""), nil
}