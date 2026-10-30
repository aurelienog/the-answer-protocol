package commands

import (
	"errors"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type GroupCommand struct{}

func NewGroupCommand() *GroupCommand {
	return &GroupCommand{}
}

func (c *GroupCommand) Name() string {
	return "GROUP"
}

func (c *GroupCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	if len(ctx.RawArgs) == 0 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	switch strings.ToUpper(ctx.RawArgs[0]) {
	case "CREATE":
		if len(ctx.RawArgs) < 2 {
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
		}

		groupName := strings.TrimSpace(strings.Join(ctx.RawArgs[1:], " "))
		if groupName == "" {
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
		}

		groupID, err := ctx.World.CreateGroup(ctx.PlayerName, groupName)
		if err != nil {
			return mapGroupError(err), nil
		}

		return protocol.OK("group=" + groupID + " name=" + groupName), nil

	case "INVITE":
		if len(ctx.RawArgs) < 2 {
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
		}

		targetName := strings.TrimSpace(strings.Join(ctx.RawArgs[1:], " "))
		if targetName == "" {
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
		}

		if err := ctx.World.InvitePlayer(ctx.PlayerName, targetName); err != nil {
			return mapGroupError(err), nil
		}

		return protocol.OK(""), nil

	case "INVITATIONS":
		if len(ctx.RawArgs) != 1 {
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
		}

		invitations, err := ctx.World.GetGroupInvitations(ctx.PlayerName)
		if err != nil {
			return mapGroupError(err), nil
		}

		return protocol.OKJSON(invitations)

	case "JOIN":
		if len(ctx.RawArgs) < 2 {
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
		}

		groupName := strings.TrimSpace(strings.Join(ctx.RawArgs[1:], " "))
		if groupName == "" {
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
		}

		joinedGroupID, err := ctx.World.JoinGroup(ctx.PlayerName, groupName)
		if err != nil {
			return mapGroupError(err), nil
		}

		return protocol.OK("group=" + joinedGroupID), nil

	case "LEAVE":
		if len(ctx.RawArgs) != 1 {
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
		}

		if err := ctx.World.LeaveGroup(ctx.PlayerName); err != nil {
			return mapGroupError(err), nil
		}

		return protocol.OK(""), nil

	default:
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}
}

func mapGroupError(err error) protocol.Message {
	switch {
	case errors.Is(err, world.ErrInvalidArgument):
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS")
	case errors.Is(err, world.ErrPlayerAlreadyInGroup):
		return protocol.ERR(protocol.CodeAlreadyInGroup, "ALREADY_IN_GROUP")

	case errors.Is(err, world.ErrPlayerNotInGroup):
		return protocol.ERR(protocol.CodeNotInGroup, "NOT_IN_GROUP")

	case errors.Is(err, world.ErrInvalidWorldState):
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR")

	case errors.Is(err, world.ErrGroupAlreadyExists):
		return protocol.ERR(protocol.CodeCommandFailed, "GROUP_ALREADY_EXISTS")

	case errors.Is(err, world.ErrPlayerNotFound),
		errors.Is(err, world.ErrGroupNotFound),
		errors.Is(err, world.ErrGroupInvitationNotFound),
		errors.Is(err, world.ErrGroupInvitationAlreadyExists):
		return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED")

	default:
		return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED")
	}
}