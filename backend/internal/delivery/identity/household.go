package identity

import (
	"net/http"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

// Closed family enrollment shares this server's session and ceremony boundary.
func (s *Server) registerHouseholdRoutes() {
	s.mux.HandleFunc("GET /api/v1/household", s.household)
	s.mux.HandleFunc("GET /api/v1/household/invitations", s.invitations)
	s.mux.HandleFunc("POST /api/v1/household/invitations", s.issueInvitation)
	s.mux.HandleFunc("POST /api/v1/household/invitations/{id}/revoke", s.revokeInvitation)
	s.mux.HandleFunc("POST /api/v1/invitations/preview", s.previewInvitation)
	s.mux.HandleFunc("POST /api/v1/invitations/accept", s.acceptInvitation)
}

func (s *Server) household(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authorize(w, r, false)
	if !ok {
		return
	}
	d, err := s.service.Household(r.Context(), a.Token)
	if err != nil {
		s.problem(w, err)
		return
	}
	members := make([]generated.HouseholdMember, 0, len(d.Members))
	for _, m := range d.Members {
		status := generated.MembershipStatus("pending")
		if m.Membership.Active {
			status = "active"
		}
		members = append(members, generated.HouseholdMember{User: generated.User{Id: string(m.User.ID), Name: m.User.Name}, Membership: generated.Membership{Id: string(m.Membership.ID), HouseholdId: string(m.Membership.HouseholdID), UserId: string(m.Membership.UserID), Role: "member", Status: status}})
	}
	s.write(w, 200, generated.Household{Id: string(d.Household.ID), Name: d.Household.Name, Timezone: d.Timezone, MaxActiveMembers: d.Maximum, Members: members})
}

func (s *Server) invitationStateDTO(state household.InvitationState) generated.InvitationState {
	result := generated.InvitationState{Revision: generated.Revision(state.Revision)}
	if i := state.Current; i != nil {
		result.Current = &generated.Invitation{Id: i.ID, InvitedBy: string(i.InvitedBy), ExpiresAt: i.ExpiresAt.UTC().Format(time.RFC3339Nano), Status: generated.InvitationStatus(i.Status)}
	}
	return result
}

func (s *Server) invitations(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authorize(w, r, false)
	if !ok {
		return
	}
	state, err := s.service.Invitations(r.Context(), a.Token)
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, s.invitationStateDTO(state))
}

func (s *Server) issueInvitation(w http.ResponseWriter, r *http.Request) {
	var input generated.InvitationMutationInput
	if !s.decode(w, r, "InvitationMutationInput", &input) {
		return
	}
	if _, ok := s.authorize(w, r, true); !ok {
		return
	}
	request, err := s.request(w, r, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	result, err := s.service.IssueInvitation(r.Context(), request, uint64(input.ExpectedRevision))
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, generated.InvitationCreated{State: s.invitationStateDTO(result.State), InvitationToken: string(result.Token)})
}

func (s *Server) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	var input generated.InvitationMutationInput
	if !s.decode(w, r, "InvitationMutationInput", &input) {
		return
	}
	if _, ok := s.authorize(w, r, true); !ok {
		return
	}
	request, err := s.request(w, r, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	state, err := s.service.RevokeInvitation(r.Context(), request, r.PathValue("id"), uint64(input.ExpectedRevision))
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, s.invitationStateDTO(state))
}

func (s *Server) previewInvitation(w http.ResponseWriter, r *http.Request) {
	var input generated.InvitationTokenInput
	if !s.decode(w, r, "InvitationTokenInput", &input) {
		return
	}
	request, err := s.request(w, r, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	preview, err := s.service.PreviewInvitation(r.Context(), request, identity.Token(*input.InvitationToken))
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, generated.InvitationPreview{HouseholdName: preview.HouseholdName, InviterName: preview.InviterName, ExpiresAt: preview.ExpiresAt.UTC().Format(time.RFC3339Nano)})
}

func (s *Server) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	var input generated.InvitationAcceptInput
	if !s.decode(w, r, "InvitationAcceptInput", &input) {
		return
	}
	request, err := s.request(w, r, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	c := input.Credential
	transports := []string{}
	if c.Transports != nil {
		for _, t := range c.Transports {
			transports = append(transports, string(t))
		}
	}
	result, err := s.service.CompleteInvitation(r.Context(), request, identity.Token(*input.InvitationToken), input.EnrollmentAttemptId, input.CredentialName, application.Registration{ID: c.Id, RawID: c.RawId, ClientDataJSON: c.ClientDataJSON, AttestationObject: c.AttestationObject, Transports: transports})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.cookie(w, result.Access)
	s.write(w, 200, generated.InitialEnrollmentResult{Purpose: "invitation", Me: s.meDTO(result.Access), RecoveryCodes: generated.RecoveryCodes{Codes: result.Codes}})
}
