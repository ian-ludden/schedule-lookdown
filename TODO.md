# TODOs for Ad-Hoc Group Schedules
- Steps toward building an improved ad-hoc group schedules tool.
- Listed in suggested order of tackling.
- Detailed plan with review checkpoints: see [AD_HOC_PLAN.md](AD_HOC_PLAN.md).
- Chosen input approach: a filterable multi-select picker built on `bubbles/list` ([Bubbles repo](https://github.com/charmbracelet/bubbles)). It needs the username directory first, hence the order below.

## 1. Spike: inspect the live ad-hoc group page
- Look at how the current ad-hoc group schedules page structures its request (URL, params, response shape).
- Look at the username dropdown markup (option values, "Last, First" format, size, whether a term param is needed).

## 2. Fetch and parse the username directory
- Grab the list from the dropdown on the ad-hoc group schedules page.
  - Sure, it's somewhat long, but should still be reasonable to retrieve and store temporarily rather than trying to store it permanently and periodically detect changes.
- Keep it in memory only (fetch once after auth, like the term list).

## 3. Multi-select picker with filtering (`bubbles/list`)
- Replaces the old "Add multiple username inputs" step; Bubbles has no multi-value text input, so a list picker is used instead.
- Absorbs the old "Auto-complete usernames and show Last, First" step: list built-in filtering, with "Last, First" shown next to the username.
- Absorbs the old "Fuzzy-match usernames/full names" step: list filtering is fuzzy; match on username or first/last name.
- Selections persist across filter changes; the submitted value is just the usernames, for easy use in GET request(s).

## 4. Retrieve the group grid for all selected people (a single GET)
- Can probably reuse the GET request structure for individual schedules.
- Use the request structure found in step 1.
- Confirmed in the spike: one `GET reg-sched.pl?type=gr_Grid&termcode=T&view=tgrid&id6=<ids space-separated>` returns the merged grid, so no parallel per-person fetching is needed.

## 5. Visualize the schedules for all selected usernames
- Try a few different viz options, see what feels best. Get user input!
- It would be nice to see which entries are from which people.
  - Color-code?
  - Add symbols? (!, *, #, &, ^, dagger, etc.)

## 6. (Optional) Support quick import/copy-paste of comma-separated usernames
- To make it easy to quickly repeat past queries, consider adding an alternate search screen where a user can paste in a comma-separated list of usernames.
- This alternate view should then just populate the picker with those usernames.
- It will be important to validate those usernames against the directory from step 2 and warn the user if any are not found.
