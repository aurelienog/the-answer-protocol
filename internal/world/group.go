package world

import (
	"fmt"
	"sort"
	"strings"
)

// CreateGroup creates a group with the player as its first member.
func (w *World) CreateGroup(playerName, groupName string) (string, error) {
	if w == nil {
		return "", fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return "", fmt.Errorf("%w: player name cannot be empty", ErrInvalidArgument)
	}

	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return "", fmt.Errorf("%w: group name cannot be empty", ErrInvalidArgument)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.playerRooms[playerName]; !exists {
		return "", fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	if _, exists := w.playerGroups[playerName]; exists {
		return "", fmt.Errorf("%w: %q", ErrPlayerAlreadyInGroup, playerName)
	}

	if w.groups == nil {
		w.groups = make(map[string]Group)
	}
	if w.playerGroups == nil {
		w.playerGroups = make(map[string]string)
	}

	for _, group := range w.groups {
		if strings.EqualFold(group.Name, groupName) {
			return "", fmt.Errorf("%w: group name %q already exists", ErrGroupAlreadyExists, groupName)
		}
	}

	var groupID string
	for {
		w.nextGroupID++
		groupID = fmt.Sprintf("group:%d", w.nextGroupID)

		if _, exists := w.groups[groupID]; !exists {
			break
		}
	}

	w.groups[groupID] = Group{
		ID:          groupID,
		Name:        groupName,
		Members:     []string{playerName},
		Invitations: make(map[string]bool),
	}
	w.playerGroups[playerName] = groupID

	return groupID, nil
}

// InvitePlayer invites a registered player to the inviter's group.
func (w *World) InvitePlayer(inviterName, targetName string) (GroupInvite, error) {
	if w == nil {
		return GroupInvite{}, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	inviterName = strings.TrimSpace(inviterName)
	targetName = strings.TrimSpace(targetName)

	if inviterName == "" || targetName == "" {
		return GroupInvite{}, fmt.Errorf(
			"%w: inviter name and target name cannot be empty",
			ErrInvalidArgument,
		)
	}

	if inviterName == targetName {
		return GroupInvite{}, fmt.Errorf("%w: a player cannot invite themselves", ErrInvalidArgument)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.playerRooms[inviterName]; !exists {
		return GroupInvite{}, fmt.Errorf("%w: %q", ErrPlayerNotFound, inviterName)
	}

	if _, exists := w.playerRooms[targetName]; !exists {
		return GroupInvite{}, fmt.Errorf("%w: %q", ErrPlayerNotFound, targetName)
	}

	groupID, exists := w.playerGroups[inviterName]
	if !exists {
		return GroupInvite{}, fmt.Errorf("%w: inviter %q", ErrPlayerNotInGroup, inviterName)
	}

	group, exists := w.groups[groupID]
	if !exists {
		return GroupInvite{}, fmt.Errorf(
			"%w: player %q references nonexistent group %q",
			ErrInvalidWorldState,
			inviterName,
			groupID,
		)
	}

	if _, exists := w.playerGroups[targetName]; exists {
		return GroupInvite{}, fmt.Errorf("%w: %q", ErrPlayerAlreadyInGroup, targetName)
	}

	if group.Invitations == nil {
		group.Invitations = make(map[string]bool)
	}

	if group.Invitations[targetName] {
		return GroupInvite{}, fmt.Errorf(
			"%w: player %q",
			ErrGroupInvitationAlreadyExists,
			targetName,
		)
	}

	group.Invitations[targetName] = true
	w.groups[groupID] = group

	return GroupInvite{
		GroupID:     groupID,
		GroupName:   group.Name,
		Inviter: inviterName,
		Invitee: targetName,
	}, nil
}

// JoinGroup joins a group when the player has a pending invitation.
func (w *World) JoinGroup(playerName, groupName string) (string, GroupJoin, error) {
	if w == nil {
		return "", GroupJoin{}, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	groupName = strings.TrimSpace(groupName)
	playerName = strings.TrimSpace(playerName)

	if playerName == "" || groupName == "" {
		return "", GroupJoin{}, fmt.Errorf(
			"%w: player name and group name cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.playerRooms[playerName]; !exists {
		return "", GroupJoin{}, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	if _, exists := w.playerGroups[playerName]; exists {
		return "", GroupJoin{}, fmt.Errorf("%w: %q", ErrPlayerAlreadyInGroup, playerName)
	}

	var group Group
	groupID := ""

	for id, candidate := range w.groups {
		if strings.EqualFold(candidate.Name, groupName) {
			group = candidate
			groupID = id
			break
		}
	}

	if groupID == "" {
		return "", GroupJoin{}, fmt.Errorf("%w: %q", ErrGroupNotFound, groupName)
	}

	if !group.Invitations[playerName] {
		return "", GroupJoin{},  fmt.Errorf(
			"%w: player %q for group %q",
			ErrGroupInvitationNotFound,
			playerName,
			groupID,
		)
	}

	members := make([]string, 0, len(group.Members))

	for _, member := range group.Members {
		members = append(members, member)
	}

	group.Members = append(group.Members, playerName)
	delete(group.Invitations, playerName)

	w.groups[groupID] = group
	w.playerGroups[playerName] = groupID

	return groupID, GroupJoin{
		GroupID:    groupID,
		GroupName:  group.Name,
		PlayerName: playerName,
		Members:    members,
	}, nil
}

// LeaveGroup removes a player from their current group.
func (w *World) LeaveGroup(playerName string) (GroupLeave, error) {
	if w == nil {
		return GroupLeave{}, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return GroupLeave{}, fmt.Errorf("%w: player name cannot be empty", ErrInvalidArgument)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.playerRooms[playerName]; !exists {
		return GroupLeave{}, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	if _, exists := w.playerGroups[playerName]; !exists {
		return GroupLeave{}, fmt.Errorf("%w: %q", ErrPlayerNotInGroup, playerName)
	}

	groupID := w.playerGroups[playerName]
	group, exists := w.groups[groupID]
	if !exists {
		return GroupLeave{}, fmt.Errorf(
			"%w: player %q references nonexistent group %q",
			ErrInvalidWorldState,
			playerName,
			groupID,
		)
	}

	members := make([]string, 0, len(group.Members))

	for _, member := range group.Members {
		if member != playerName {
			members = append(members, member)
		}
	}

	event := GroupLeave{
		GroupID:    groupID,
		GroupName:  group.Name,
		PlayerName: playerName,
		Members:    members,
	}

	if err := w.removePlayerFromGroupLocked(playerName); err != nil {
		return GroupLeave{}, err
	}

	return event, nil
}

// removePlayerFromGroupLocked removes a player from their group.
// The caller must hold w.mu.Lock().
func (w *World) removePlayerFromGroupLocked(playerName string) error {
	groupID, exists := w.playerGroups[playerName]
	if !exists {
		return nil
	}

	group, exists := w.groups[groupID]
	if !exists {
		return fmt.Errorf(
			"%w: player %q references nonexistent group %q",
			ErrInvalidWorldState,
			playerName,
			groupID,
		)
	}

	members := make([]string, 0, len(group.Members))
	found := false

	for _, member := range group.Members {
		if member == playerName {
			found = true
			continue
		}
		members = append(members, member)
	}

	if !found {
		return fmt.Errorf(
			"%w: group %q does not contain player %q",
			ErrInvalidWorldState,
			groupID,
			playerName,
		)
	}

	delete(w.playerGroups, playerName)

	if len(members) == 0 {
		delete(w.groups, groupID)
		return nil
	}

	group.Members = members
	w.groups[groupID] = group

	return nil
}

// GetGroupInvitations returns the groups that have invited a player.
func (w *World) GetGroupInvitations(playerName string) ([]GroupInvitation, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return nil, fmt.Errorf("%w: player name cannot be empty", ErrInvalidArgument)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.playerRooms[playerName]; !exists {
		return nil, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	invitations := make([]GroupInvitation, 0)

	for _, group := range w.groups {
		if group.Invitations[playerName] {
			invitations = append(invitations, GroupInvitation{
				ID:   group.ID,
				Name: group.Name,
			})
		}
	}

	sort.Slice(invitations, func(i, j int) bool {
		return strings.ToLower(invitations[i].Name) < strings.ToLower(invitations[j].Name)
	})

	return invitations, nil
}