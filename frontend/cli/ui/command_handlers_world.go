package ui

import (
    "encoding/json"
)

func handleLookResponse(m *Model, payload string) error {
    var response LookResponse

    if err := json.Unmarshal([]byte(payload), &response); err != nil {
        return err
    }

    m.world.Room = response
    m.setView(ViewRoom)

    return nil
}

func handleMoveResponse(m *Model, payload string) error {
    return nil
}

//take, drop, talk, quest 
