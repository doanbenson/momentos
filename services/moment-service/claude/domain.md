# moment-service — domain

Owns write-path for these glossary terms (see root `GLOSSARY.md`):

**Moment** · **Cut** · **Style Profile** · **Vibe** · **Song Pick** · **Week Intake** (as the command that creates a Moment)

**Editor** commands this service accepts (MVP): reorder Assets, replace Asset, remove Asset, trim a clip source window, request re-draft / re-render. Trim, reorder, replace, and remove update the stored Cut in place. cut-draft is enqueued only when reDraft is set.

Does not own: **Asset** bytes, **Edit Audio** files, **Render** files.
