package autopull

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/github"
	"github.com/parkr/auto-reply/ctx"
	"github.com/parkr/auto-reply/hooks"
)

var (
	baseBranch *string = github.String("master")
)

func shortMessage(message string) string {
	return strings.SplitN(message, "\n", 1)[0]
}

// branchFromRef takes "refs/heads/pull/my-pull" and returns "pull/my-pull"
func branchFromRef(ref string) string {
	return strings.Replace(ref, "refs/heads/", "", 1)
}

func prBodyForPush(push *github.PushEvent) string {
	var mention string
	if author := push.Commits[0].Author; author != nil {
		if author.Login != nil {
			mention = *author.Login
		} else {
			mention = *author.Name
		}
	} else {
		mention = "unknown"
	}
	return fmt.Sprintf(
		"PR automatically created for @%s.\n\n```text\n%s\n```",
		mention,
		*push.Commits[0].Message,
	)
}

func newPRForPush(push *github.PushEvent) *github.NewPullRequest {
	if push.Commits == nil || len(push.Commits) == 0 {
		return nil
	}
	return &github.NewPullRequest{
		Title: github.String(shortMessage(*push.Commits[0].Message)),
		Head:  github.String(branchFromRef(*push.Ref)),
		Base:  github.String("master"),
		Body:  github.String(prBodyForPush(push)),
	}
}

func AutomaticallyCreatePullRequest(repos ...string) hooks.EventHandler {
	puller := autoPuller{repos: repos}
	return puller.Handler
}

type autoPuller struct {
	repos []string
}

func (h *autoPuller) handlesRepo(repo string) bool {
	for _, handled := range h.repos {
		if handled == repo {
			return true
		}
	}
	return false
}

func (h *autoPuller) Handler(context *ctx.Context, event interface{}) error {
	push, ok := event.(*github.PushEvent)
	if !ok {
		return context.NewError("AutoPull: not an push event")
	}

	if strings.HasPrefix(*push.Ref, "refs/heads/pull/") && h.handlesRepo(*push.Repo.FullName) {
		pr := newPRForPush(push)
		if pr == nil {
			return context.NewError("AutoPull: no commits for %s on %s/%s", *push.Ref, *push.Repo.Owner.Name, *push.Repo.Name)
		}

		pull, _, err := context.GitHub.PullRequests.Create(*push.Repo.Owner.Name, *push.Repo.Name, pr)
		if err != nil {
			return context.NewError(
				"AutoPull: error creating pull request for %s on %s/%s: %v",
				*push.Ref, *push.Repo.Owner.Name, *push.Repo.Name, err,
			)
		}
		log.Println("created pull request: %s#%d", *push.Repo.FullName, *pull.Number)
	}

	return nil
}
