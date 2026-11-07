# Demo credits

Every photograph, engraving, text and data set the demo shows, with the page it was verified on. Files are in `assets/media/`. All were downloaded and re-encoded for this pass; nothing is AI generated. Licences are quoted as the source page states them.

## Photographs from Pexels

Pexels pages show "License: Free" linked to https://www.pexels.com/license/, which says "All photos and videos on Pexels are free to use" and "Attribution is not required". Credit is given anyway.

| File in the demo | Title | Photographer | Source page | What changed |
| --- | --- | --- | --- | --- |
| cover-observatory.webp | Dream | Horacio Lander | https://www.pexels.com/photo/dream-27796994/ | Cropped to 10:3, resized to 1600x480, WebP, metadata stripped |
| cover-milkyway.webp | Starry Night Sky in Kyrgyzstan's Mountains | Daniel Torobekov (alohaphotostudio) | https://www.pexels.com/photo/starry-night-sky-in-kyrgyzstan-s-mountains-39876995/ | Cropped to 10:3, resized to 1600x480, WebP, metadata stripped |
| cover-telescope.webp, telescope-milky-way.webp | Sky Filled With Stars at Night | Lucas Pezeta | https://www.pexels.com/photo/sky-filled-with-stars-at-night-2106759/ | Cover cropped to 10:3 at 1600x480; inline copy resized to 1600 px wide; WebP, metadata stripped |
| cover-tripod.webp, telescope-tripod.webp | Starry Night Sky | Lucas Pezeta | https://www.pexels.com/photo/starry-night-sky-7387427/ | Cover cropped to 10:3 at 1600x480; inline copy resized to 1600 px wide; WebP, metadata stripped |
| moon-full-mono.webp | Close-Up Shot of a Full Moon | Yilin Mo | https://www.pexels.com/photo/close-up-shot-of-a-full-moon-13190293/ | Trimmed black margin, resized to 1100 px, WebP, metadata stripped |
| moon-full-warm.webp | Full Moon Against Black Night Sky | tommy picone (tommaso) | https://www.pexels.com/photo/full-moon-against-black-night-sky-30968622/ | Trimmed black margin, resized to 1100 px high, WebP, metadata stripped |

## NASA images

Source: the NASA Image and Video Library record for each item (https://images-api.nasa.gov/search?nasa_id=ID, shown at https://images.nasa.gov/details/ID). The record pages carry no licence text, so the licence is taken from NASA's media guidelines page https://www.nasa.gov/nasa-brand-center/images-and-media/, which says NASA content "generally are not subject to copyright in the United States". None of the pictures shows a NASA insignia or an identifiable person. Captions on the Moon missions page are the NASA record descriptions, shortened.

| File in the demo | NASA ID, title, date | Credit | What changed |
| --- | --- | --- | --- |
| artemis-ii-earthset.webp, cover-earthset.webp | art002e021278, A Breathtaking Earthset from Orion, 2026-04-06 | NASA | Resized to 1600 px wide; the cover is a 10:3 crop at 1600x480; WebP |
| apollo-8-earthrise.webp | as08-14-2383, Apollo 8 Mission image, Earth over the horizon of the moon, 1968-12-24 | NASA | Resized to 1100 px, WebP |
| apollo-11-saturn-v.webp | 8909250, Saturn Apollo Program, 1969-07-16 | NASA, Marshall Space Flight Center | Resized to 1000 px high, WebP |
| apollo-11-bootprint.webp | as11-40-5877, Apollo 11 Mission image, Astronaut bootprint on the lunar surface, 1969-07-20 | NASA | Resized to 1100 px, WebP |

## Galileo, The Sidereal Messenger

Text and engravings: Galileo Galilei, Sidereus Nuncius (Venice, 1610), as translated by Edward Stafford Carlos (1880), Project Gutenberg ebook 46036, https://www.gutenberg.org/ebooks/46036. The page states "Public domain in the USA." Engravings come from https://www.gutenberg.org/cache/epub/46036/images/ (p016a.jpg, p016b.jpg, p072.png). The text on the Galileo page is two passages, joined into paragraphs, with margin notes, page numbers and footnote marks removed and two dashes replaced by colons. The Project Gutenberg trademark header and licence text are not reproduced. Files: galileo-moon-sketch-1.webp, galileo-moon-sketch-2.webp, galileo-jupiter-moons.webp. Changed: converted to WebP at the original size.

## Data

- exoplanets.js: 66 planets from the NASA Exoplanet Archive, table `pscomppars`, queried through its TAP service (https://exoplanetarchive.ipac.caltech.edu/TAP). The archive's acknowledgement page (https://exoplanetarchive.ipac.caltech.edu/docs/acknowledge.html) asks that work using it state: "This research has made use of the NASA Exoplanet Archive, which is operated by the California Institute of Technology, under contract with the National Aeronautics and Space Administration under the Exoplanet Exploration Program." That page gives no licence text. The values are measurements published by NASA's archive; the demo reproduces them with that acknowledgement. Changes: rows chosen by name, columns renamed, distance converted from parsecs to light-years (x 3.26156), numbers rounded, and a Class column derived from radius (under 1.25 Earth-size, under 2 Super-Earth, under 6 Neptune-like, else Gas giant).
- Galilean moons table: NASA Space Science Data Coordinated Archive, Jovian Satellite Fact Sheet, https://nssdc.gsfc.nasa.gov/planetary/factsheet/joviansatfact.html, bulk parameters for Io, Europa, Ganymede and Callisto, copied unchanged. Treated as NASA content under the same media guidelines.
- Apollo 11 mission objective sentence: https://www.nasa.gov/mission/apollo-11/, NASA.

## Original text

The notebook welcome page, the Dark sky weekend page and the table of targets are written for the demo. Distances in the targets table are rounded literature values.
