package ui

import (
    "the-answer-protocol/frontend/cli/network"
    "the-answer-protocol/internal/protocol"
)

type commandResultMsg struct {
    command  string
    response protocol.Message
    err      error
}

type authenticationResultMsg struct {
    username string
    err      error
}

type serverEventMsg struct {
    event network.IncomingMessage
}

type clientDoneMsg struct{}