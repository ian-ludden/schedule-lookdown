# Sample responses from reg-sched.pl / reg-download.pl

All names, usernames, Banner IDs, and emails in these files are **fake** —
synthetic test data, not real people. They mirror the structure of real
reg-sched.pl responses so the parser and `--load-samples` mode can be exercised
without authenticating or shipping anyone's data. Course codes, titles, rooms,
and times are retained for realism.

- [Instructor schedule](./sample-instructor.html)
    - URL: `https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl?type=Instructor&termcode=202630&view=tgrid&id=valer`
- [Course roster](./sample-roster.html)
    - URL: `https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl?type=Roster&termcode=202630&view=tgrid&id=CSSE474-02`
- [Course roster: combined sections](./sample-roster-combined.html)
    - URL: `https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl?type=Roster&termcode=202630&view=tgrid&id=CSSE474`
- [Student schedule](./sample-student.html)
    - URL: `https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl?type=Username&termcode=202630&view=tgrid&id=quinna`
- [Course list](./sample-courses.html)
    - URL: `https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl?type=Course&termcode=202630&view=tgrid&id=CSSE3`
- [Lastname Search](./sample-lastname-search.html)
    - URL: `https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl?termcode=202710&view=tgrid&lnameid=Vale&lnamebt=Lastname&id1=&id4=&id5=&deptid=`
- [Downloaded roster (CSV)](./CSSE220-01.csv)
    - The CSV returned by the roster page's "Download Roster" button.
    - URL: `https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-download.pl` (POST, form fields `id=CSSE220-01` and `download=Download Roster`)
