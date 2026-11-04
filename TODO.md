# TODOs for Ad-Hoc Group Schedules
- Steps toward building an improved ad-hoc group schedules tool.
- Listed in suggested order of tackling. 

## Add multiple username inputs
- See if Bubble Tea already has built-in support for this type of input field. 
- If not, then decide between comma-separated usernames or a fixed number of username fields (with the option of leaving some blank). 

## Retrieve the schedules for all selected usernames
- Can probably reuse the GET request structure for individual schedules. 
- Worth taking a look at how the current ad-hoc group schedules page structures this request. 
- Maybe this is a time to try out Golang's coroutines, for retrieving different people's schedules in parallel? 

## Visualize the schedules for all selected usernames
- Try a few different viz options, see what feels best. Get user input! 
- It would be nice to see which entries are from which people. 
  - Color-code? 
  - Add symbols? (!, *, #, &, ^, dagger, etc.)

## Auto-complete usernames and show "Last, First"
- Grab the list from the dropdown on the ad-hoc group schedules page. 
  - Sure, it's somewhat long, but should still be reasonable to retrieve and store temporarily rather than trying to store it permanently and periodically detect changes. 
- Look for a built-in/library way of filtering; fairly common use-case, so maybe Bubble Tea already solved this problem. 
- Display the name off to the side; keep the actual field text as just the username for easy use in GET request(s). 

## (Optional) Fuzzy-match usernames/full names
- Look for a built-in/library tool for fuzzy-match filtering as an upgrade to exact-match filtering. 
- Ideally, fuzzy-match either username or first/last name

## (Optional) Support quick import/copy-paste of comma-separated usernames
- To make it easy to quickly repeat past queries, consider adding an alternate search screen where a user can paste in a comma-separated list of usernames. 
- This alternate view should then just populate the default view with those sernames. 
- It will be important to validate those usernames and warn the user if any are not found. 

