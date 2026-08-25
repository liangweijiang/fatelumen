package service

import (
	"context"
	"errors"
	"testing"

	"fatelumen/backend/internal/birthchart"
)

type fixedProfileLocationResolver struct{}

func (fixedProfileLocationResolver) Version() string { return "test" }
func (fixedProfileLocationResolver) Search(context.Context, string, string) ([]birthchart.LocationCandidate, error) {
	return nil, nil
}
func (fixedProfileLocationResolver) Resolve(_ context.Context, input birthchart.LocationInput) (*birthchart.ResolvedLocation, error) {
	if input.PlaceID != "1812545" {
		return nil, birthchart.ErrInvalidLocation
	}
	return &birthchart.ResolvedLocation{CountryCode: "CN", CountryName: "China", RegionCode: "30", RegionName: "Dongguan", City: "Dongguan", PlaceID: input.PlaceID, DisplayName: "Dongguan, China", Longitude: 113.74866, Latitude: 23.01797, TimezoneID: input.TimezoneID}, nil
}

func validProfileInput(name string) CreateProfileInput {
	return CreateProfileInput{
		DisplayName: name, Gender: 1, CalendarType: 0,
		BirthYear: 1992, BirthMonth: 3, BirthDay: 15,
		BirthHour: 8, BirthMinute: 30, Timezone: "Asia/Shanghai",
	}
}

func TestProfileValidationAcceptsCompleteReportInput(t *testing.T) {
	if err := validateProfileInput(validProfileInput("report subject")); err != nil {
		t.Fatalf("complete input rejected: %v", err)
	}
	unknownHour := validProfileInput("unknown hour")
	unknownHour.BirthHour = -1
	if err := validateProfileInput(unknownHour); err != nil {
		t.Fatalf("unknown hour should remain valid: %v", err)
	}
}

func TestProfileValidationRejectsIncompleteOrImpossibleInput(t *testing.T) {
	cases := []CreateProfileInput{
		{Gender: 1, CalendarType: 0, BirthYear: 1992, BirthMonth: 2, BirthDay: 31, BirthHour: 8, Timezone: "Asia/Shanghai"},
		{Gender: 1, CalendarType: 0, BirthYear: 1992, BirthMonth: 3, BirthDay: 15, BirthHour: 8},
		{Gender: 1, CalendarType: 0, BirthYear: 1992, BirthMonth: 3, BirthDay: 15, BirthHour: 8, Timezone: "Mars/Olympus"},
		{Gender: 2, CalendarType: 0, BirthYear: 1992, BirthMonth: 3, BirthDay: 15, BirthHour: 8, Timezone: "Asia/Shanghai"},
	}
	for i, in := range cases {
		if err := validateProfileInput(in); !errors.Is(err, ErrInvalidProfile) {
			t.Fatalf("case %d expected ErrInvalidProfile, got %v", i, err)
		}
	}
}

func TestProfileServiceCanonicalizesLocationBeforePersistence(t *testing.T) {
	svc := &ProfileService{locations: fixedProfileLocationResolver{}}
	in := validProfileInput("subject")
	in.PlaceID = "1812545"
	in.Timezone = "Asia/Shanghai"
	if err := svc.resolveLocation(context.Background(), &in); err != nil {
		t.Fatalf("resolveLocation() error = %v", err)
	}
	if in.Longitude != 113.74866 || in.Latitude != 23.01797 || !in.HasCoordinates || in.City != "Dongguan" {
		t.Fatalf("location was not canonicalized: %+v", in)
	}
}
