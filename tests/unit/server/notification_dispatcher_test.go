package server_test

import (
	"testing"

	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

func TestNotificationDispatcherEncodesAndRoutesEvents(t *testing.T) {
	tests := []struct {
		name     string
		event    any
		expected map[string]string
	}{
		{
			name: "player enters room",
			event: world.PlayerEntered{
				PlayerName:  "Alice",
				RoomPlayers: []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Bob": "EVT ROOM PRESENCE ENTER Alice\n",
			},
		},
		{
			name: "player leaves room",
			event: world.PlayerLeft{
				PlayerName:  "Alice",
				RoomPlayers: []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Bob": "EVT ROOM PRESENCE LEAVE Alice\n",
			},
		},
		{
			name: "player movement notifies each room",
			event: world.PlayerMoved{
				PlayerName:          "Alice",
				PreviousRoomPlayers: []string{"Alice", "Bob"},
				NewRoomPlayers:      []string{"Alice", "Carol"},
			},
			expected: map[string]string{
				"Bob":   "EVT ROOM PRESENCE LEAVE Alice\n",
				"Carol": "EVT ROOM PRESENCE ENTER Alice\n",
			},
		},
		{
			name: "item dropped",
			event: world.ItemDropped{
				PlayerName:  "Alice",
				ItemID:      "item.rope",
				RoomPlayers: []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Bob": "EVT ITEM_DROPPED item.rope\n",
			},
		},
		{
			name: "unique items returned to everyone",
			event: world.UniqueItemsReturned{
				ItemIDs: []string{"instance:item.sword:1", "instance:item.shield:2"},
			},
			expected: map[string]string{
				"Alice": "EVT UNIQUE_ITEMS_RETURNED instance:item.sword:1,instance:item.shield:2\n",
				"Bob":   "EVT UNIQUE_ITEMS_RETURNED instance:item.sword:1,instance:item.shield:2\n",
				"Carol": "EVT UNIQUE_ITEMS_RETURNED instance:item.sword:1,instance:item.shield:2\n",
			},
		},
		{
			name: "empty unique items return emits nothing",
			event: world.UniqueItemsReturned{
				ItemIDs: []string{},
			},
			expected: map[string]string{},
		},
		{
			name: "group invite",
			event: world.GroupInvite{
				GroupName: "Red Team",
				Inviter:   "Alice",
				Invitee:   "Bob",
			},
			expected: map[string]string{
				"Bob": "EVT GROUP INVITE player=Alice group=Red Team\n",
			},
		},
		{
			name: "group join",
			event: world.GroupJoin{
				PlayerName: "Alice",
				Members:    []string{"Alice", "Bob", "Carol"},
			},
			expected: map[string]string{
				"Bob":   "EVT GROUP JOIN player=Alice\n",
				"Carol": "EVT GROUP JOIN player=Alice\n",
			},
		},
		{
			name: "group leave",
			event: world.GroupLeave{
				PlayerName: "Alice",
				Members:    []string{"Alice", "Bob", "Carol"},
			},
			expected: map[string]string{
				"Bob":   "EVT GROUP LEAVE player=Alice\n",
				"Carol": "EVT GROUP LEAVE player=Alice\n",
			},
		},
		{
			name: "chat",
			event: world.ChatMessage{
				Scope:      "GLOBAL",
				PlayerName: "Alice",
				Message:    "hello",
				Recipients: []string{"Alice", "Bob", "offline"},
			},
			expected: map[string]string{
				"Alice": "EVT GLOBAL CHAT player=Alice message=hello\n",
				"Bob":   "EVT GLOBAL CHAT player=Alice message=hello\n",
			},
		},
		{
			name: "NPC death",
			event: world.ParticipantDied{
				ParticipantName: "Dire Wolf",
				KillerID:        "Alice",
				Recipients:      []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Alice": "EVT COMBAT NPC_DIED participant=Dire Wolf killer=Alice\n",
				"Bob":   "EVT COMBAT NPC_DIED participant=Dire Wolf killer=Alice\n",
			},
		},
		{
			name: "player death",
			event: world.ParticipantDied{
				ParticipantName: "Alice",
				IsPlayer:        true,
				KillerID:        "dire_wolf",
				Recipients:      []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Alice": "EVT COMBAT PLAYER_DIED participant=Alice killer=dire_wolf\n",
				"Bob":   "EVT COMBAT PLAYER_DIED participant=Alice killer=dire_wolf\n",
			},
		},
		{
			name: "combat started",
			event: world.CombatStarted{
				CombatID:     "combat-1",
				RoomID:       "docks",
				Participants: []string{"Alice", "dock_warden"},
				Recipients:   []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Alice": "EVT COMBAT STARTED combat=combat-1 room=docks participants=Alice,dock_warden\n",
				"Bob":   "EVT COMBAT STARTED combat=combat-1 room=docks participants=Alice,dock_warden\n",
			},
		},
		{
			name: "player attack",
			event: world.CombatAttack{
				AttackerName: "Alice",
				TargetName:   "Dock Warden",
				IsPlayer:     true,
				Recipients:   []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Alice": "EVT COMBAT ATTACK attacker_type=PLAYER attacker=Alice target=Dock Warden\n",
				"Bob":   "EVT COMBAT ATTACK attacker_type=PLAYER attacker=Alice target=Dock Warden\n",
			},
		},
		{
			name: "NPC attack",
			event: world.CombatAttack{
				AttackerName: "Dock Warden",
				TargetName:   "Alice",
				Recipients:   []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Alice": "EVT COMBAT ATTACK attacker_type=NPC attacker=Dock Warden target=Alice\n",
				"Bob":   "EVT COMBAT ATTACK attacker_type=NPC attacker=Dock Warden target=Alice\n",
			},
		},
		{
			name: "combat damage",
			event: world.CombatDamage{
				AttackerName: "Alice",
				TargetName:   "Dock Warden",
				Damage:       12,
				RemainingHP:  28,
				Recipients:   []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Alice": "EVT COMBAT DAMAGE attacker=Alice target=Dock Warden amount=12 remaining_hp=28\n",
				"Bob":   "EVT COMBAT DAMAGE attacker=Alice target=Dock Warden amount=12 remaining_hp=28\n",
			},
		},
		{
			name: "player fled",
			event: world.PlayerFled{
				PlayerName:  "Alice",
				Damage:      5,
				RemainingHP: 35,
				Recipients:  []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Bob": "EVT COMBAT PLAYER_FLED player=Alice damage=5 remaining_hp=35\n",
			},
		},
		{
			name: "combat turn changed",
			event: world.CombatTurnChanged{
				ParticipantName:  "Dock Warden",
				Round:            2,
				AvailableActions: 1,
				Recipients:       []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Alice": "EVT COMBAT TURN participant_type=NPC participant=Dock Warden round=2 actions=1\n",
				"Bob":   "EVT COMBAT TURN participant_type=NPC participant=Dock Warden round=2 actions=1\n",
			},
		},
		{
			name: "player combat turn changed",
			event: world.CombatTurnChanged{
				ParticipantName:  "Alice",
				IsPlayer:         true,
				Round:            2,
				AvailableActions: 2,
				Recipients:       []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Alice": "EVT COMBAT TURN participant_type=PLAYER participant=Alice round=2 actions=2\n",
				"Bob":   "EVT COMBAT TURN participant_type=PLAYER participant=Alice round=2 actions=2\n",
			},
		},
		{
			name: "combat help requested",
			event: world.CombatHelpRequested{
				PlayerName: "Alice",
				CombatID:   "combat-1",
				RoomID:     "docks",
				Recipients: []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Bob": "EVT COMBAT HELP_REQUESTED player=Alice combat=combat-1 room=docks\n",
			},
		},
		{
			name: "player joined combat",
			event: world.CombatPlayerJoined{
				PlayerName: "Alice",
				CombatID:   "combat-1",
				Recipients: []string{"Alice", "Bob"},
			},
			expected: map[string]string{
				"Alice": "EVT COMBAT PLAYER_JOINED player=Alice combat=combat-1\n",
				"Bob":   "EVT COMBAT PLAYER_JOINED player=Alice combat=combat-1\n",
			},
		},
		{
			name: "item used with HP change",
			event: world.ItemUsed{
				PlayerName: "Alice",
				ItemID:     "item.healing_potion",
				StatChanges: map[string]int{
					world.CombatStatHP: 40,
				},
			},
			expected: map[string]string{
				"Alice": "EVT ITEM_USED player=Alice item=item.healing_potion hp=40\n",
			},
		},
		{
			name: "item used without HP change",
			event: world.ItemUsed{
				PlayerName: "Alice",
				ItemID:     "item.iron_sword",
			},
			expected: map[string]string{
				"Alice": "EVT ITEM_USED player=Alice item=item.iron_sword\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := server.NewRegistry()
			sessions := map[string]*server.Session{
				"Alice": newRegistryTestSession("session-alice", "Alice"),
				"Bob":   newRegistryTestSession("session-bob", "Bob"),
				"Carol": newRegistryTestSession("session-carol", "Carol"),
			}
			for _, session := range sessions {
				if err := registry.Add(session); err != nil {
					t.Fatalf("register %q: %v", session.PlayerName, err)
				}
			}

			dispatcher := server.NewNotificationDispatcher(registry)
			dispatcher.Handle(tt.event)

			for name, session := range sessions {
				want, shouldReceive := tt.expected[name]
				if !shouldReceive {
					if got := len(session.Outbox); got != 0 {
						t.Errorf("%s received %d unexpected notifications", name, got)
					}
					continue
				}

				if got := len(session.Outbox); got != 1 {
					t.Fatalf("%s received %d notifications, want 1", name, got)
				}
				if got := <-session.Outbox; got != want {
					t.Errorf("%s notification = %q, want %q", name, got, want)
				}
			}
		})
	}
}
