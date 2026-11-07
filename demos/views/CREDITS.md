# Demo credits: views

Every photograph and data set the views demo shows. Photographs are in `assets/media/`. All were downloaded from NASA and re-encoded for the demo; nothing is AI generated, nothing is from a stock site. Licences are quoted as the source states them.

## NASA photographs

Source for every row: the NASA Image and Video Library record, shown at `https://images.nasa.gov/details/ID` and read as JSON at `https://images-api.nasa.gov/search?nasa_id=ID` (each record was re-read on 2026-11-07 and its `nasa_id` and title match the table). The record pages carry no licence text. The licence is taken from NASA's media guidelines, https://www.nasa.gov/nasa-brand-center/images-and-media/, which says NASA content "generally are not subject to copyright in the United States" and that the NASA insignia and logotype "are not in the public domain". None of the photographs below shows a NASA insignia as its subject, and none shows an identifiable person as its subject: they are launches, spacecraft, recovery scenes and views of Earth and the Moon. The Gemini 3 mating picture shows a pad technician in clean-room garb, not identifiable.

Credit for all rows: NASA. Centre as listed by the record: JSC (Johnson Space Center), MSFC (Marshall Space Flight Center), KSC (Kennedy Space Center). What changed in every row: resized to at most 900 px on the long side and converted to WebP.

| File in the demo | NASA ID | Title in the record | Date | Centre |
| --- | --- | --- | --- | --- |
| mr-3.webp | 6100884 | Mercury Project (launch of Freedom 7) | 1961-05-05 | MSFC |
| mr-4.webp | 6116424 | Mercury Project (helicopter and Liberty Bell 7) | 1961-07-21 | MSFC |
| ma-6.webp | 9139490 | Mercury Project (launch of Friendship 7) | 1962-02-20 | MSFC |
| ma-7.webp | 62PC-52 | KSC-62pc-52 (Aurora 7 liftoff) | 1962-05-24 | KSC |
| ma-8.webp | S62-06006 | LIFTOFF - "SIGMA 7" - MERCURY-ATLAS (MA)-8 - CAPE | 1962-10-03 | JSC |
| ma-9.webp | s63-09630 | MA-9 - SPLASHDOWN | 1963-05-16 | JSC |
| gt-3.webp | S65-17337 | MATING - GEMINI-TITAN (GT)-3 - CAPE | 1965-02-17 | JSC |
| gt-4.webp | S65-29639 | View of the Gemini-Titan 4 spacecraft launch | 1965-06-03 | JSC |
| gt-5.webp | s65-50774 | Launch of the Gemini 5 spacecraft from Pad 19 | 1965-08-21 | JSC |
| gt-6a.webp | S65-63171 | View of the Gemini 6 and Gemini 7 rendezvous | 1965-12-15 | JSC |
| gt-7.webp | s65-64040 | Gemini-Titan (GT)-7 of GT-6 Space Photography | 1965-12-15 | JSC |
| gt-8.webp | S66-24482 | LIFTOFF - GEMINI-TITAN (GT)-8 - ATLAS/AGENA - CAPE | 1966-03-16 | JSC |
| gt-9a.webp | s66-37923 | GEMINI-TITAN (GT)-9 - EARTH-SKY - AUGMENTED TARGET DOCKING ADAPTER (ATDA) | 1966-06-06 | JSC |
| gt-10.webp | s66-46249 | GEMINI-TITAN (GT)-10 - EARTH SKY - DOCKING - IN-SPACE | 1966-07-18 | JSC |
| gt-11.webp | s66-54563 | Gulf Coast from Atchafalaya Bay to Apalachicola seen from Gemini 11 | 1966-09-14 | JSC |
| gt-12.webp | 9141925 | Gemini 12 launch on a Titan vehicle | 1966-11-11 | MSFC |
| ap-7.webp | S68-48662 | APOLLO VII - LAUNCH - KSC | 1968-10-11 | JSC |
| ap-8.webp | as08-14-2383 | Apollo 8 Mission image, Earth over the horizon of the moon | 1968-12-24 | JSC |
| ap-9.webp | as09-24-3652 | Apollo 9 Mission image - Command Module | 1969-03-03 | JSC |
| ap-10.webp | as10-34-5013 | Apollo 10 view of the Earth | 1969-05-18 | JSC |
| ap-11.webp | 6900556 | Saturn Apollo Program (Apollo 11 launch) | 1969-07-16 | MSFC |
| ap-12.webp | 6903391 | Saturn Apollo Program (Apollo 12 liftoff) | 1969-11-14 | MSFC |
| ap-13.webp | as13-59-8501 | View of damaged Apollo 13 Service Module | 1970-04-17 | JSC |
| ap-14.webp | 7120033 | Saturn Apollo Program (Apollo 14 launch) | 1971-01-31 | MSFC |
| ap-15.webp | as15-87-11849 | View of Mount Hadley as photographed by Apollo 15 | 1971-07-31 | JSC |
| ap-16.webp | S72-35613 | INFLIGHT - APOLLO 16 (LM LAUNCH) | 1972-04-23 | JSC |
| ap-17.webp | as17-148-22727 | View of the Earth seen by the Apollo 17 crew | 1972-12-07 | JSC |

The cover picture, `cover-views.webp`, is a 1600x350 crop of the Earth from as08-14-2383 (the Apollo 8 record above) set on black, so the cover has the same source and licence as `ap-8.webp`. Changes: cropped around the Earth, background set to black, WebP.

## Data

The Crewed missions, Astronauts and Programs tables were typed from public NASA records of the 27 US crewed flights from Freedom 7 to Apollo 17. Mission dates, flight times, orbits, crews, launch vehicles and recovery areas come from:

- NASA SP-4002, Appendix 8, "Manned Space Flight Record Summary of Mercury and Gemini", https://www.nasa.gov/history/SP-4002/app8.htm. The page is an image of a table, credited "From Mercury and Gemini Mission Reports", and carries no copyright notice.
- NASA Space Science Data Coordinated Archive, "The Gemini Program (1962 - 1966)", https://nssdc.gsfc.nasa.gov/planetary/gemini.html.
- NASA Space Science Data Coordinated Archive chronology of US astronaut missions 1961 to 1972, for the Apollo flights and launch dates.

These are facts published by NASA; they are treated as NASA content under the same media guidelines. Derived columns (Days, Crew-hours, Length, Years ago, Veteran, per-astronaut flight counts and hours, per-program rollups) are computed by the demo's formulas and rollups from those values. Short mission notes ("First orbital.") are written for the demo.

## Original text

The page title, the lead sentences and the program notes are written for the demo.
