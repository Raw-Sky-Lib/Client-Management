> **Canonical source:** See `Agency-Hub/.claude/LINEAR-SETUP.md` for the full issue list.
> **Historical reference.** Most ACH issues are complete. See `Agency-Hub/.claude/TODO.md`.

# agency-hub — Linear Project (Summary)

```
Team:       agency-hub
Identifier: ACH (issues: ACH-5 through ACH-50)
Status:     Core build complete (ACH-5 to ACH-44 done)
            QA/E2E items (ACH-45 to ACH-50) incomplete
```

## Completed Milestones

- M1: Foundation & Auth (backend) — done
- M2: Client Management (backend) — done (management_token dropped, portal invite added)
- M3: Team Management (backend) — done
- M4: Projects & Deploy (backend) — done (deploy moved to projects, no deploy_records)
- M5: Dashboard — done
- M6-M10: All Frontend — done

## Key Architecture Changes Since Original Issues

| Original plan | What was actually built |
|---------------|------------------------|
| `management_token` on clients | Dropped entirely (migration 015) |
| `deploy_records` table | Dropped; deploy on `projects` (migration 018) |
| Project kanban status | Dropped (migration 017) |
| Connection token = primary onboarding | Now legacy; portal invite = primary path |
| Client-level Supabase credentials | Now on projects (migration 017/018) |

See `Agency-Hub/.claude/TODO.md` for full ACH issue checklist with completion dates.
