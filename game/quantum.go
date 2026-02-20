package game

import "fmt"

// SuperpositionGroup holds all board squares where a single quantum piece exists simultaneously.
type SuperpositionGroup struct {
	ID      int
	Squares []Square
	Piece   Piece // canonical piece info (type, color)
}

// EntanglementGroup represents pieces that are entangled: capturing one destroys all others.
type EntanglementGroup struct {
	ID      int
	Squares []Square    // one entry per member piece
	Edges   [][2]Square // adjacency list of link pairs
}

// QuantumSplit: the piece at `from` stays as one superposed instance; a ghost appears at `to`.
// Both squares must be reachable via non-capturing moves; `to` must be empty.
// Creates or extends a SuperpositionGroup.
func (b *Board) QuantumSplit(from, to Square) error {
	p := b.piece(from)
	if p == nil {
		return fmt.Errorf("no piece at %v", from)
	}
	if p.Color != b.Turn {
		return fmt.Errorf("not your piece")
	}
	if p.Type == King {
		return fmt.Errorf("king cannot be split")
	}
	if b.piece(to) != nil {
		return fmt.Errorf("destination must be empty for quantum split")
	}
	// Verify `to` is reachable via a non-capturing move from `from`.
	reachable := b.LegalSplitTargets(from)
	found := false
	for _, sq := range reachable {
		if sq == to {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("destination %v not reachable for split", to)
	}

	// Determine superposition group ID.
	sid := p.SuperpositionID
	if sid == 0 {
		// Create new group; register `from` as first square.
		sid = b.nextSuperID
		b.nextSuperID++
		b.SuperpositionGroups[sid] = &SuperpositionGroup{
			ID:      sid,
			Squares: []Square{from},
			Piece:   *p,
		}
		p.SuperpositionID = sid
	}

	// Add `to` as a new ghost.
	ghost := &Piece{
		Type:            p.Type,
		Color:           p.Color,
		SuperpositionID: sid,
		EntanglementIDs: append([]int(nil), p.EntanglementIDs...),
	}
	b.Cells[to.Row][to.Col] = ghost
	b.SuperpositionGroups[sid].Squares = append(b.SuperpositionGroups[sid].Squares, to)

	// Advance turn.
	b.advanceTurn()
	return nil
}

// Collapse resolves a superposition group by randomly (deterministically for tests: first) choosing one square.
// The chosen instance becomes classical; ghosts are removed.
// Returns the chosen Square.
func (b *Board) Collapse(groupID int) Square {
	sg, ok := b.SuperpositionGroups[groupID]
	if !ok || len(sg.Squares) == 0 {
		return Square{}
	}
	// Choose first square deterministically (caller may override for UI randomness).
	chosen := sg.Squares[0]
	b.collapseToSquare(groupID, chosen)
	return chosen
}

// CollapseToSquare collapses a superposition group to a specific square.
func (b *Board) CollapseToSquare(groupID int, chosen Square) {
	b.collapseToSquare(groupID, chosen)
}

func (b *Board) collapseToSquare(groupID int, chosen Square) {
	sg, ok := b.SuperpositionGroups[groupID]
	if !ok {
		return
	}
	// Remove all ghost squares except chosen.
	for _, sq := range sg.Squares {
		if sq == chosen {
			// Make it classical.
			if p := b.piece(sq); p != nil {
				p.SuperpositionID = 0
			}
		} else {
			b.Cells[sq.Row][sq.Col] = nil
		}
	}
	delete(b.SuperpositionGroups, groupID)
}

// Link adds an entanglement edge between two friendly pieces (current player).
// Only one link/unlink allowed per turn.
func (b *Board) Link(a, bSq Square) error {
	if b.LinkedThisTurn {
		return fmt.Errorf("already used link/unlink action this turn")
	}
	pa := b.piece(a)
	pb := b.piece(bSq)
	if pa == nil || pb == nil {
		return fmt.Errorf("both squares must contain pieces")
	}
	if pa.Color != b.Turn || pb.Color != b.Turn {
		return fmt.Errorf("can only link your own pieces")
	}
	if a == bSq {
		return fmt.Errorf("cannot link a piece to itself")
	}

	// Find existing groups for each square.
	groupA := b.groupContaining(a)
	groupB := b.groupContaining(bSq)

	if groupA == nil && groupB == nil {
		// Create new group.
		id := b.nextEntangleID
		b.nextEntangleID++
		eg := &EntanglementGroup{
			ID:      id,
			Squares: []Square{a, bSq},
			Edges:   [][2]Square{{a, bSq}},
		}
		b.EntanglementGroups[id] = eg
		b.addEntanglementID(a, id)
		b.addEntanglementID(bSq, id)
	} else if groupA == nil {
		// Add `a` to groupB.
		groupB.Squares = append(groupB.Squares, a)
		groupB.Edges = append(groupB.Edges, [2]Square{a, bSq})
		b.addEntanglementID(a, groupB.ID)
	} else if groupB == nil {
		// Add `bSq` to groupA.
		groupA.Squares = append(groupA.Squares, bSq)
		groupA.Edges = append(groupA.Edges, [2]Square{a, bSq})
		b.addEntanglementID(bSq, groupA.ID)
	} else if groupA.ID != groupB.ID {
		// Merge groupB into groupA.
		for _, sq := range groupB.Squares {
			groupA.Squares = append(groupA.Squares, sq)
			b.removeEntanglementID(sq, groupB.ID)
			b.addEntanglementID(sq, groupA.ID)
		}
		groupA.Edges = append(groupA.Edges, groupB.Edges...)
		groupA.Edges = append(groupA.Edges, [2]Square{a, bSq})
		delete(b.EntanglementGroups, groupB.ID)
	} else {
		// Same group: just add edge (if not duplicate).
		groupA.Edges = append(groupA.Edges, [2]Square{a, bSq})
	}

	b.LinkedThisTurn = true
	return nil
}

// Unlink removes one entanglement edge between two squares.
// If the removal disconnects the graph, splits into separate EntanglementGroups.
// Only one link/unlink allowed per turn.
func (b *Board) Unlink(a, bSq Square) error {
	if b.LinkedThisTurn {
		return fmt.Errorf("already used link/unlink action this turn")
	}
	eg := b.groupContaining(a)
	if eg == nil {
		return fmt.Errorf("no entanglement group contains %v", a)
	}

	// Remove the edge.
	newEdges := eg.Edges[:0:0]
	removed := false
	for _, e := range eg.Edges {
		if (e[0] == a && e[1] == bSq) || (e[0] == bSq && e[1] == a) {
			if !removed {
				removed = true
				continue
			}
		}
		newEdges = append(newEdges, e)
	}
	if !removed {
		return fmt.Errorf("no link between %v and %v", a, bSq)
	}
	eg.Edges = newEdges

	// Check connectivity; split if needed.
	components := b.findComponents(eg)
	if len(components) > 1 {
		// Remove old group.
		delete(b.EntanglementGroups, eg.ID)
		for _, sq := range eg.Squares {
			b.removeEntanglementID(sq, eg.ID)
		}
		// Create new groups per component.
		for _, comp := range components {
			if len(comp.Squares) < 2 {
				// Lone piece: no group needed.
				continue
			}
			id := b.nextEntangleID
			b.nextEntangleID++
			comp.ID = id
			b.EntanglementGroups[id] = comp
			for _, sq := range comp.Squares {
				b.addEntanglementID(sq, id)
			}
		}
	}

	b.LinkedThisTurn = true
	return nil
}

// TriggerEntanglement: called after a piece at `sq` is confirmed captured.
// Removes all other pieces in any entanglement group that included `sq`.
func (b *Board) TriggerEntanglement(sq Square) {
	p := b.piece(sq)
	if p == nil {
		return
	}
	// Collect all group IDs for this piece.
	gids := append([]int(nil), p.EntanglementIDs...)
	for _, gid := range gids {
		eg, ok := b.EntanglementGroups[gid]
		if !ok {
			continue
		}
		for _, msq := range eg.Squares {
			if msq == sq {
				continue
			}
			// Remove this member piece.
			mp := b.piece(msq)
			if mp != nil {
				// If it's in superposition, collapse first.
				if mp.SuperpositionID != 0 {
					// Just wipe all squares of that group.
					if sg, ok := b.SuperpositionGroups[mp.SuperpositionID]; ok {
						for _, ssq := range sg.Squares {
							b.Cells[ssq.Row][ssq.Col] = nil
						}
						delete(b.SuperpositionGroups, mp.SuperpositionID)
					}
				} else {
					b.Cells[msq.Row][msq.Col] = nil
				}
			}
		}
		delete(b.EntanglementGroups, gid)
	}
}

// --- helpers ---

func (b *Board) groupContaining(sq Square) *EntanglementGroup {
	p := b.piece(sq)
	if p == nil {
		return nil
	}
	for _, gid := range p.EntanglementIDs {
		if eg, ok := b.EntanglementGroups[gid]; ok {
			return eg
		}
	}
	return nil
}

func (b *Board) addEntanglementID(sq Square, id int) {
	p := b.piece(sq)
	if p == nil {
		return
	}
	for _, eid := range p.EntanglementIDs {
		if eid == id {
			return
		}
	}
	p.EntanglementIDs = append(p.EntanglementIDs, id)
}

func (b *Board) removeEntanglementID(sq Square, id int) {
	p := b.piece(sq)
	if p == nil {
		return
	}
	eids := p.EntanglementIDs[:0]
	for _, eid := range p.EntanglementIDs {
		if eid != id {
			eids = append(eids, eid)
		}
	}
	p.EntanglementIDs = eids
}

// findComponents does BFS/DFS over the edges of eg and returns connected components.
func (b *Board) findComponents(eg *EntanglementGroup) []*EntanglementGroup {
	// Build adjacency map.
	adj := make(map[Square][]Square)
	for _, sq := range eg.Squares {
		adj[sq] = nil
	}
	for _, e := range eg.Edges {
		adj[e[0]] = append(adj[e[0]], e[1])
		adj[e[1]] = append(adj[e[1]], e[0])
	}

	visited := make(map[Square]bool)
	var components []*EntanglementGroup

	for _, start := range eg.Squares {
		if visited[start] {
			continue
		}
		// BFS.
		comp := &EntanglementGroup{}
		queue := []Square{start}
		visited[start] = true
		sqSet := make(map[Square]bool)
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			comp.Squares = append(comp.Squares, cur)
			sqSet[cur] = true
			for _, nb := range adj[cur] {
				if !visited[nb] {
					visited[nb] = true
					queue = append(queue, nb)
				}
			}
		}
		// Collect edges within this component.
		for _, e := range eg.Edges {
			if sqSet[e[0]] && sqSet[e[1]] {
				comp.Edges = append(comp.Edges, e)
			}
		}
		components = append(components, comp)
	}
	return components
}

// advanceTurn switches the active player and resets per-turn flags.
func (b *Board) advanceTurn() {
	if b.Turn == Black {
		b.FullMoveNumber++
	}
	b.Turn = opponent(b.Turn)
	b.LinkedThisTurn = false
}
