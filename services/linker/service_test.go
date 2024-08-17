package linker

import (
	"context"
	"fmt"
	"testing"
	"time"
	"vcassist-backend/lib/testutil"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"
	"vcassist-backend/services/linker/db"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
)

var testWeightKeys = []string{
	"Intro to Computer Prog (CodeHS)",
	"Hip Hop II",
	"English 10 (H)",
	"Trig/Pre-Calculus AB",
	"Photographic Design Studio (H)",
	"US History",
	"Theatre III",
	"Dance Technique III (H)",
	"American Literature",
	"English 9",
	"AP Statistics",
	"Art of Filmmaking",
	"Foundations of Water Polo",
	"Aerospace Engineering",
	"ASB Student Government",
	"Indoor Sports Performance (Girls)",
	"Outdoor Sports Performance (G)",
	"Theatre II",
	"Ceramics I",
	"AP Microeconomics",
	"Service Learning: Justice",
	"Dance Technique I",
	"Hip Hop III (H)",
	"Trig/Pre-Calculus BC (H)",
	"Integrated Science",
	"Physics",
	"Intro to Engineering Design",
	"Hip Hop III",
	"Economics",
	"English 10",
	"String Ensemble",
	"REACH II",
	"History and Music",
	"Personal Finance & Stewardship",
	"Foundations of Baseball",
	"Lifetime Fitness (Girls)",
	"American Sign Language I",
	"Debate",
	"English 9 (H)",
	"Chemistry (H)",
	"Anatomy/Physiology (H)",
	"AP Physics 1",
	"AP Chinese Language and Culture",
	"Advanced Applied Filmmaking (H)",
	"Biology (H)",
	"Algebra I",
	"Graphic Design",
	"Guitar II",
	"Indoor Sports Performance (Boys)",
	"Spanish I",
	"Principles of Engineering",
	"Algebra II",
	"Geometry (H)",
	"Applied Filmmaking",
	"3D Animation",
	"Piano I",
	"Dance Technique III",
	"AP Music Theory",
	"Modern World History",
	"AP Latin",
	"AP US History",
	"Yearbook",
	"Mandarin IV (H)",
	"AP US Government & Politics",
	"Modern World History (H)",
	"iPad Tech Support Internship",
	"Speech",
	"Dance Technique II",
	"AP Calculus BC",
	"Guitar III",
	"Intro to Guitar & Electric Bass",
	"Latin III",
	"AP Calculus AB",
	"Percussion Ensemble",
	"AP Human Geography",
	"Adv Design & Production Intern",
	"Entrepreneurship (H)",
	"PE Extension",
	"AP Chemistry",
	"Marine Biology",
	"Astronomy",
	"Art III (H)",
	"American Sign Language II",
	"AP Comp Sci Principles (CodeHS)",
	"AP English Language and Comp",
	"Algebra II (H)",
	"Multi-Variable Calculus",
	"Jazz Ensemble (H)",
	"US Government",
	"Service Learning: Missions",
	"Tech Theatre III (H)",
	"Art II",
	"Spanish IV",
	"British Literature",
	"Calculus",
	"Anatomy/Physiology",
	"Biology",
	"Worship Choir",
	"Piano III",
	"AP Biology",
	"Ceramics III (H)",
	"Mandarin I",
	"Business Fundamentals",
	"AP Computer Science A (CodeHS)",
	"Hip Hop I",
	"Conservatory Chorus",
	"Theatre I",
	"AP Spanish Language & Culture",
	"AP African American Studies",
	"Ceramics II",
	"Latin III (H)",
	"French II",
	"Wisdom for Leaders",
	"Service Learning: Mentoring",
	"Christian Practice & Belief",
	"Scientific Research",
	"Advanced Data Analysis",
	"REACH",
	"Spanish II",
	"French IV (H)",
	"AP English Literature and Comp",
	"Statistics",
	"Wind Ensemble",
	"Lifetime Fitness (Boys)",
	"Theatre IV (H)",
	"Latin II",
	"Christianity & Leadership",
	"Outdoor Sports Performance (B)",
	"AP Environmental Science",
	"Spanish III",
	"AP Art History",
	"AP Studio Art: Drawing",
	"African American Literature",
	"Geometry",
	"Mathematics of Financial Analysis",
	"Piano II",
	"Jazz Lab",
	"Chemistry",
	"Mandarin III",
	"Philosophy in Literature",
	"Football Performance",
	"AP Physics C: Electricity & Magnetism",
	"Art I",
	"Service Learning: Outreach",
	"AP Physics C: Mechanics",
	"Photographic Design I",
	"Spanish III (H)",
	"New Testament Studies",
	"Philosophy of Religion",
	"Chamber Ensemble",
	"Mandarin II",
	"American Sign Language IV (H)",
	"Latin I",
	"Trigonometry/Pre-Calculus",
	"Traditional Animation Techniques",
	"Tech Theatre I",
	"Tech Theatre II",
	"AP Studio Art: 2D Design",
	"French III",
	"Advanced 3D Animation",
	"American Sign Language III",
	"French I",
}

func TestService(t *testing.T) {
	res, cleanup := testutil.SetupService(t, testutil.ServiceParams{
		Name:     "services/linker",
		DbSchema: db.Schema,
	})
	defer cleanup()
	service := NewService(res.DB)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	{
		res, err := service.GetKnownKeys(ctx, &connect.Request[linkerv1.GetKnownKeysRequest]{
			Msg: &linkerv1.GetKnownKeysRequest{},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, len(res.Msg.GetKeys()), 0, "expected no known keys")
	}
	{
		res, err := service.GetKnownSets(ctx, &connect.Request[linkerv1.GetKnownSetsRequest]{
			Msg: &linkerv1.GetKnownSetsRequest{},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, len(res.Msg.GetSets()), 0, "expected no known sets")
	}
	{
		res, err := service.GetExplicitLinks(ctx, &connect.Request[linkerv1.GetExplicitLinksRequest]{
			Msg: &linkerv1.GetExplicitLinksRequest{
				LeftSet:  "random set",
				RightSet: "random set 2",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, len(res.Msg.GetLeftKeys()), 0, "expected no explicit links to exist")
		require.Equal(t, len(res.Msg.GetRightKeys()), 0, "expected no explicit links to exist")
	}

	_, err := service.AddExplicitLink(ctx, &connect.Request[linkerv1.AddExplicitLinkRequest]{
		Msg: &linkerv1.AddExplicitLinkRequest{
			Left: &linkerv1.ExplicitKey{
				Set: "powerschool",
				Key: "Physics 1 (H)",
			},
			Right: &linkerv1.ExplicitKey{
				Set: "moodle",
				Key: "Physics 1",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddExplicitLink(ctx, &connect.Request[linkerv1.AddExplicitLinkRequest]{
		Msg: &linkerv1.AddExplicitLinkRequest{
			Left: &linkerv1.ExplicitKey{
				Set: "moodle",
				Key: "Physics 1 Honors",
			},
			Right: &linkerv1.ExplicitKey{
				Set: "powerschool",
				Key: "Physics 1 (H)",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	linkRes, err := service.Link(ctx, &connect.Request[linkerv1.LinkRequest]{
		Msg: &linkerv1.LinkRequest{
			Src: &linkerv1.Set{
				Name: "weights",
				Keys: testWeightKeys,
			},
			Dst: &linkerv1.Set{
				Name: "powerschool",
				Keys: []string{
					"Unscheduled",
					"Chapel",
					"AP Statistics",
					"AP US Government & Politics",
					"Data Structures and Algorithms (H)",
					"Philosophy In Literature (H)",
					"AP Physics C: Mechanics",
					"Multi-Variable Calculus (H)",
					"Philosophy of Religion",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(linkRes.Msg.String())
	diff := cmp.Diff(
		map[string]string{
			"AP Physics C: Mechanics":     "AP Physics C: Mechanics",
			"AP Statistics":               "AP Statistics",
			"AP US Government & Politics": "AP US Government & Politics",
			"Philosophy of Religion":      "Philosophy of Religion",
		},
		linkRes.Msg.GetSrcToDst(),
	)
	if diff != "" {
		t.Fatal(diff)
	}

	suggestRes, err := service.SuggestLinks(ctx, &connect.Request[linkerv1.SuggestLinksRequest]{
		Msg: &linkerv1.SuggestLinksRequest{
			SetLeft:  "weights",
			SetRight: "powerschool",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	diff = cmp.Diff(
		[]*linkerv1.LinkSuggestion{
			{
				LeftKey:  "Multi-Variable Calculus",
				RightKey: "Multi-Variable Calculus (H)",
			},
			{
				LeftKey:  "Philosophy in Literature",
				RightKey: "Philosophy In Literature (H)",
			},
		},
		suggestRes.Msg.GetSuggestions(),
		cmpopts.IgnoreUnexported(linkerv1.LinkSuggestion{}),
	)
	if diff != "" {
		t.Fatal(diff)
	}
}
