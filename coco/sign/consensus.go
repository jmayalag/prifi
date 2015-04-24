package sign

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/dedis/prifi/coco/coconet"
)

func (sn *Node) SetupProposal(view int, am *AnnouncementMessage, from string) error {
	// if this is for viewchanges: otherwise new views are not allowed
	if am.Vote.Type == ViewChangeVT {
		// viewchange votes must be received from the new parent on the new view
		if view != am.Vote.Vcv.View {
			log.Errorln("recieved view change vote on different view")
			return errors.New("view change attempt on view != received view")
		}
		// ensure that we are caught up
		if int(sn.LastAppliedVote) != sn.LastSeenVote {
			log.Errorln(sn.Name(), "received vote: but not up to date: need to catch up", sn.LastAppliedVote, sn.LastSeenVote)
			return errors.New("not up to date: need to catch up")
		}
		if sn.RootFor(am.Vote.Vcv.View) != am.Vote.Vcv.Root {
			log.Errorln("received vote: but invalid root", sn.RootFor(am.Vote.Vcv.View), am.Vote.Vcv.Root)
			return errors.New("invalid root for proposed view")
		}

		nextview := sn.ViewNo + 1
		for ; nextview <= view; nextview++ {
			sn.NewViewFromPrev(nextview, from)
			for _, act := range sn.Actions[nextview] {
				sn.ApplyAction(nextview, act)
			}
		}
		fmt.Fprintln(os.Stderr, sn.Name(), "setuppropose:", sn.HostListOn(view))
		fmt.Fprintln(os.Stderr, sn.Name(), "setuppropose:", sn.Parent(view))
	} else {
		if view != sn.ViewNo {
			log.Errorln("cannot vote on not-current view")
			return errors.New("vote on not-current view")
		}
	}

	if am.Vote.Type == AddVT {
		if am.Vote.Av.View <= sn.ViewNo {
			log.Errorln("cannot change current-view or previous views")
			return errors.New("unable to change past views")
		}
	}
	if am.Vote.Type == RemoveVT {
		if am.Vote.Rv.View <= sn.ViewNo {
			log.Errorln("cannot change current-view or previous views")
			return errors.New("unable to change past views")
		}
	}
	return nil
}

// A propose for a view change would come on current view + sth
// when we receive view change  message on a future view,
// we must be caught up, create that view  and apply actions on it
func (sn *Node) Propose(view int, am *AnnouncementMessage, from string) error {
	log.Println(sn.Name(), "GOT ", "Propose", am)
	if err := sn.SetupProposal(view, am, from); err != nil {
		return err
	}

	if err := sn.setUpRound(view, am); err != nil {
		return err
	}
	// log.Println(sn.Name(), "propose on view", view, sn.HostListOn(view))
	sn.Rounds[am.Round].Vote = am.Vote

	// Inform all children of proposal
	messgs := make([]coconet.BinaryMarshaler, sn.NChildren(view))
	for i := range messgs {
		sm := SigningMessage{
			Type:         Announcement,
			View:         view,
			LastSeenVote: sn.LastSeenVote,
			Am:           am}
		messgs[i] = &sm
	}
	ctx := context.TODO()
	//ctx, _ := context.WithTimeout(context.Background(), 2000*time.Millisecond)
	if err := sn.PutDown(ctx, view, messgs); err != nil {
		return err
	}

	if len(sn.Children(view)) == 0 {
		log.Println(sn.Name(), "no children")
		sn.Promise(view, am.Round, nil)
	}
	return nil
}

func (sn *Node) Promise(view, Round int, sm *SigningMessage) error {
	log.Println(sn.Name(), "GOT ", "Promise", sm)
	// update max seen round
	sn.LastSeenRound = max(sn.LastSeenRound, Round)

	round := sn.Rounds[Round]
	if round == nil {
		// was not announced of this round, should retreat
		return nil
	}
	if sm != nil {
		round.Commits = append(round.Commits, sm)
	}

	if len(round.Commits) != len(sn.Children(view)) {
		return nil
	}

	// cast own vote
	sn.AddVotes(Round, round.Vote)

	for _, sm := range round.Commits {
		// count children votes
		round.Vote.Count.Responses = append(round.Vote.Count.Responses, sm.Com.Vote.Count.Responses...)
		round.Vote.Count.For += sm.Com.Vote.Count.For
		round.Vote.Count.Against += sm.Com.Vote.Count.Against

	}

	return sn.actOnPromises(view, Round)
}

func (sn *Node) actOnPromises(view, Round int) error {
	round := sn.Rounds[Round]
	var err error

	if sn.IsRoot(view) {
		sn.commitsDone <- Round

		var b []byte
		b, err = round.Vote.MarshalBinary()
		if err != nil {
			// log.Fatal("Marshal Binary failed for CountedVotes")
			return err
		}
		round.c = hashElGamal(sn.suite, b, round.Log.V_hat)
		err = sn.Accept(view, &ChallengeMessage{
			C:     round.c,
			Round: Round,
			Vote:  round.Vote})

	} else {
		// create and putup own commit message
		com := &CommitmentMessage{
			Vote:  round.Vote,
			Round: Round}

		// ctx, _ := context.WithTimeout(context.Background(), 2000*time.Millisecond)
		// log.Println(sn.Name(), "puts up promise on view", view, "to", sn.Parent(view))
		ctx := context.TODO()
		err = sn.PutUp(ctx, view, &SigningMessage{
			View:         view,
			Type:         Commitment,
			LastSeenVote: sn.LastSeenVote,
			Com:          com})
	}
	return err
}

func (sn *Node) Accept(view int, chm *ChallengeMessage) error {
	log.Println(sn.Name(), "GOT ", "Accept", chm)
	// update max seen round
	sn.LastSeenRound = max(sn.LastSeenRound, chm.Round)

	round := sn.Rounds[chm.Round]
	if round == nil {
		log.Errorln("error round is nil")
		return nil
	}

	// act on decision of aggregated votes
	// log.Println(sn.Name(), chm.Round, round.VoteRequest)
	if round.Vote != nil {
		// append vote to vote log
		// potentially initiates signing node action based on vote
		sn.actOnVotes(view, chm.Vote)
	}
	if err := sn.SendChildrenChallenges(view, chm); err != nil {
		return err
	}

	if len(sn.Children(view)) == 0 {
		sn.Accepted(view, chm.Round, nil)
	}

	return nil
}

func (sn *Node) Accepted(view, Round int, sm *SigningMessage) error {
	log.Println(sn.Name(), "GOT ", "Accepted")
	// update max seen round
	sn.LastSeenRound = max(sn.LastSeenRound, Round)

	round := sn.Rounds[Round]
	if round == nil {
		// TODO: if combined with cosi pubkey, check for round.Log.v existing needed
		// If I was not announced of this round, or I failed to commit
		return nil
	}

	if sm != nil {
		round.Responses = append(round.Responses, sm)
	}
	if len(round.Responses) != len(sn.Children(view)) {
		return nil
	}
	// TODO: after having a chance to inspect the contents of the challenge
	// nodes can raise an alarm respond by ack/nack

	if sn.IsRoot(view) {
		sn.done <- Round
	} else {
		// create and putup own response message
		rm := &ResponseMessage{
			Vote:  round.Vote,
			Round: Round}

		// ctx, _ := context.WithTimeout(context.Background(), 2000*time.Millisecond)
		ctx := context.TODO()
		return sn.PutUp(ctx, view, &SigningMessage{
			Type:         Response,
			View:         view,
			LastSeenVote: sn.LastSeenVote,
			Rm:           rm})
	}

	return nil
}

func (sn *Node) actOnVotes(view int, v *Vote) {
	// TODO: percentage of nodes for quorum should be parameter
	// Basic check to validate Vote was Confirmed, can be enhanced
	// TODO: signing node can go through list of votes and verify
	accepted := v.Count.For > 2*len(sn.HostListOn(view))/3
	var actionTaken string = "rejected"
	if accepted {
		actionTaken = "accepted"
	}

	// Report on vote decision
	if sn.IsRoot(view) {
		abstained := len(sn.HostListOn(view)) - v.Count.For - v.Count.Against
		log.Infoln("Votes FOR:", v.Count.For, "; Votes AGAINST:", v.Count.Against, "; Absteined:", abstained, actionTaken)
	}
	// Act on vote Decision
	if accepted {
		sn.VoteLog.Put(v.Index, v)
		log.Println(sn.Name(), "actOnVotes: vote", v.Index, " has been accepted")
	} else {
		v.Type = NoOpVT
		sn.VoteLog.Put(v.Index, v)
		log.Println(sn.Name(), "actOnVotes: vote", v.Index, " has been rejected")

	}
	// List out all votes
	// for _, vote := range round.CountedVotes.Votes {
	// 	log.Infoln(vote.Name, vote.Accepted)
	// }
}
