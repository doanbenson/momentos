# gateway — domain

Gateway is not a domain owner. It understands **Principal** (authenticated user from Supabase JWT) only.

**Principal**:
The signed-in user identity attached to each request after JWT validation.
_Avoid_: account, customer, profile (those belong elsewhere if introduced)

All Moment / Cut / Asset / Render language: see root `GLOSSARY.md` and the owning service’s `claude/domain.md`.
