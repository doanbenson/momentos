# Momentos

A product that turns a user's selected photos, videos, and voice into a composed Moment in a Life On Film–inspired style, then lets them refine the result.

## Language

**Momentos**:
The product name. The monorepo and user-facing brand.
_Avoid_: Momento (as an identifier), Life On Film (as the product name)

**Moment**:
One composed audiovisual piece created from a user's selected media, a Reflection, and a background song.
_Avoid_: film, reel, project, video (as the name for the whole thing)

**Cut**:
The structured, editable edit plan for a Moment (timeline, pacing, transitions, sync). The source of truth for what will render; the finished file is derived from it.
_Avoid_: edit (as a noun for the plan), timeline (alone), render plan, master_timeline (as a persisted source of truth)

**Style Profile**:
A versioned base grammar for cutting (transition types, silence handling, grade presets). Applied together with **Vibe**, which further drives pacing, sync emphasis, and Song family.
_Avoid_: filter, theme, "the AI style", Life On Film (as a system name)

**Render**:
The derived audiovisual file produced from a Cut. Not the source of truth.
_Avoid_: export (when you mean the file itself), final edit

**Asset**:
A stored media unit the user contributes — a photo, video clip, or voice recording — referenced by a Moment. System media (Edit Audio, SFX, overlays) are not Assets.
_Avoid_: file, upload, media (as the noun for one item), attachment

**Reflection**:
A spoken answer to a prompt question, stored as a voice or talking-head video Asset and used as the Act 1 hook of a Moment. A role over an Asset, not a separate media store.
_Avoid_: voiceover (alone), talking head (as the domain noun), confession

**Vibe**:
A labeled mood/energy class for a Moment. It drives Song matching and also modulates how the Cut is drafted (pacing, emphasis, sync) on top of the Style Profile.
_Avoid_: mood, energy, genre (alone), theme

**Edit Audio**:
A temporary audio reference used only to draft and beat-align the Cut. Not the legally cleared track that ships on Instagram.
_Avoid_: Song (when you mean the temp reference), soundtrack, mp3

**Platform Song**:
The Instagram/Meta music-library identity for the same track, attached at Reel upload so Meta can mute Edit Audio and replace it with the cleared version plus artist credit.
_Avoid_: Song (ambiguous), Instagram audio (as a vague phrase), catalog track

**Song**:
The paired idea of Edit Audio (for cutting) + Platform Song (for legal Reel publish). Prefer the specific term when only one side is meant.
_Avoid_: catalog (Momentos does not own Meta’s library)

**Week Intake**:
The onboarding ask for Assets from the user’s recent week — one Reflection plus photo/video clips — that become the input for one Moment.
_Avoid_: onboarding dump, album, camera roll, project

**Song Pick**:
The Song pair attached to a Moment. MVP uses one hard-coded Pick; later the system may auto-fill a Pick that the user can replace.
_Avoid_: catalog browse (MVP), playlist


**Editor**:
The post-processing UI where the user previews the Moment and changes parts of the Cut (or swaps Assets) before saving or sharing.
_Avoid_: timeline tool, CapCut, studio

**Beat Grid**:
The deterministic set of beat/onset timestamps derived from Edit Audio that legal Cut clip boundaries must snap to.
_Avoid_: tempo map (alone), waveform, BPM (alone)

**Act**:
A structural segment of a Moment’s Cut in the Life On Film Style Profile — Act 1 Reflection hook, Act 2 transition into the montage (flash/shutter), Act 3 photo/clip cycle + hero look (vintage film print) — expressed on the Cut so Render can apply segment-specific grammar.
_Avoid_: chapter, scene (as the product noun for these segments)

**Motion Preset**:
A named camera-motion choice on a Cut clip (e.g. slow push, whip punch-in, handheld drift) that Render executes; planned by drafting, not chosen in a free-form Editor motion UI.
_Avoid_: Ken Burns (as the domain noun), zoom effect, camera path

**Visual Style**:
A named look grammar on a Cut clip that Render applies (e.g. clean vertical vs Super 8 gate), distinct from Motion Preset and from the overall Style Profile id.
_Avoid_: filter, LUT (as the domain noun for this clip field), grade (alone)
