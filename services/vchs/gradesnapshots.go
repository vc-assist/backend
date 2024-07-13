package vchs

import (
	"context"
	gradesnapshotspb "vcassist-backend/services/gradesnapshots/api"
	"vcassist-backend/services/studentdata/api"

	"connectrpc.com/connect"
)

func (s Service) studentDataFromGradesnapshots(ctx context.Context, userEmail string) (*api.StudentData, error) {
	res, err := s.gradesnapshots.Pull(ctx, &connect.Request[gradesnapshotspb.PullRequest]{
		Msg: &gradesnapshotspb.PullRequest{
			User: userEmail,
		},
	})
	if err != nil {
		return nil, err
	}

	data := &api.StudentData{
		Courses: make([]*api.Course, len(res.Msg.Courses)),
	}
	for i, c := range res.Msg.Courses {
		snapshots := make([]*api.GradeSnapshot, len(c.Snapshots))
		for i, snap := range c.Snapshots {
			snapshots[i] = &api.GradeSnapshot{
				Time:  snap.Time,
				Value: snap.Value,
			}
		}
		data.Courses[i] = &api.Course{
			Name:      c.Course,
			Snapshots: snapshots,
		}
	}
	return data, nil
}
