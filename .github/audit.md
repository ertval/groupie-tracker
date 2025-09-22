# groupie-tracker

#### Functional

###### Has the requirement for the allowed packages been respected? (Reminder for this project: only [standard packages](https://golang.org/pkg/))

###### Is the data from the `artists` being used?

###### Is the data from the `locations` being used?

###### Is the data from the `dates` being used?

###### Is data from the `relations` being used?

##### Try to see the "members" for the artist/band `"Queen"`

```
    "Freddie Mercury",
    "Brian May",
    "John Daecon",
    "Roger Meddows-Taylor",
    "Mike Grose",
    "Barry Mitchell",
    "Doug Fogie"
```

###### Does it present the right "member", as above?

##### Try to see the "firstAlbum" for the artist/band `"Gorillaz"`

```
    "26-03-2001"
```

###### Does it present the right date for the "firstAlbum", as above?

##### Try to see the "locations" for the artist/band `"Travis Scott"`

```
    "santiago-chile"
    "sao_paulo-brasil"
    "los_angeles-usa"
    "houston-usa"
    "atlanta-usa"
    "new_orleans-usa"
    "philadelphia-usa"
    "london-uk"
    "frauenfeld-switzerland"
    "turku-finland"
```

###### Does it present the right "locations" as above?

##### Try to see the ""members"" for the artist/band `"Foo Fighters"`.

```
    "Dave Grohl"
    "Nate Mendel"
    "Taylor Hawkins"
    "Chris Shiflett"
    "Pat Smear"
    "Rami Jaffee"
```

###### Does it present the right members as above?

##### Try to trigger an event/action using some kind of action (ex: Clicking the mouse over a certain element, pressing a key on the keyboard, resizing or closing the browser window, a form being submitted, an error occurring, etc).

###### Does the event/action responds as expected?

###### Did the server behaved as expected?(did not crashed)

###### Does the server use the right [HTTP method](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods)?

###### Did the site run without crashing at any time?

###### Are all the pages working? (Absence of 404 page?)

###### Does the project handle [HTTP status 500 - Internal Server Errors](https://www.restapitutorial.com/httpstatuscodes.html)?

###### Is the communication between server and client well established?

###### Does the server present all the needed handlers and patterns for the http requests?

###### As an auditor, is this project up to every standard? If not, why are you failing the project?(Empty Work, Incomplete Work, Invalid compilation, Cheating, Crashing, Leaks)

#### General

###### +Does the event system run as asynchronous? (usage of go routines and channels)

###### +Is the site hosted/deployed? Can you access the website through a DNS (Domain Name System)?

#### Basic

###### +Does the project run quickly and effectively? (Favoring recursive, no unnecessary data requests, etc)

###### +Does the code obey the [good practices](../../good-practices/README.md)?

###### +Is there a test file for this code?

#### Social

###### +Did you learn anything from this project?

###### +Can it be open-sourced / be used for other sources?

###### +Would you recommend/nominate this program as an example for the rest of the school?

---

# groupie-tracker-filters


#### Functional

###### Has the requirement for the allowed packages been respected? (Reminder for this project: only [standard packages](https://golang.org/pkg/))

###### Does the project have a range [filter](https://dribbble.com/shots/1751801-Ui-Elements-Social-Network-Analytics/attachments/284260)?

###### Does the project have a check box [filter](https://dribbble.com/shots/1751801-Ui-Elements-Social-Network-Analytics/attachments/284260)?

##### Try to filter the artists/bands which the creation date is between `"1995"` and `"2000"`.

###### Did SOJA, Mamonas Assassinas, Thirty Seconds to Mars, Nickleback, NWA, Gorillaz, Linkin Park, Eminem and Coldplay appear as a result?

##### Try to filter the artists/bands that recorded their first album between `"1990"` and `"1992"`.

###### Did Pearl Jam and Red Hot Chili Peppers appear as a result?"

##### Try to filter the artists/bands that have exactly `"6"` members in their band.

###### Did Pink Floyd, Arctic Monkeys, Linkin Park and Foo Fighters appear as a result?

##### Try to filter the artists/bands that have/had concerts in `"Texas, USA"`.

###### Did R3HAB, Logic, Joyner Lucas and Twenty One Pilots appear as a result?

##### Try to filter the artists/bands which the creation date is between `"1970"` and `"2000"` and have only `"1"` member (solo artists).

###### Did Bobby McFerrins and Eminem appear as a result?

##### Try to filter the artists/bands which the creation date is after `"2010"` and recorded their first album after `"2010"`.

###### Did XXXTentacion, Juice Wrld, Alec Benjamin and Post Malone appear as a result?

##### Try to filter the artists/bands that have/had concerts in `"Washington, USA"` and have more than 3 members.

###### Did The Rolling Stones appear as a result?

##### Try to filter the artists/bands that recorded their first album between `"1980"` and `"1990"` and have a maximum of `"4"` members.

###### Did Phil Collins, Bobby McFerrins, Red Hot Chili Peppers and Metallica appear as a result?

###### Can you filter so that all the artists/bands are all shown?

###### As an auditor, is this project up to every standard? If not, why are you failing the project?(Empty Work, Incomplete Work, Invalid compilation, Cheating, Crashing, Leaks)

#### General

###### +Does the result of the filters change while you are changing the filters (is it asynchronous)?

#### Basic

###### +Does the code obey the [good practices](../../../good-practices/README.md)?

###### +Are the instructions in the website clear?

#### Social

###### +Did you learn anything from this project?

###### +Would you recommend/nominate this program as an example for the rest of the school?

---

# groupie-tracker-search-bar


#### Functional

###### Has the requirement for the allowed packages been respected? (Reminder for this project: only [standard packages](https://golang.org/pkg/))

##### Start typing in the search bar `"Billie Joe"`.

###### Does it present as suggestions the member "Billie Joe Armstrong"?

##### Start typing in the search bar `"Japan"`.

###### Does it present as suggestions the locations "saitama-japan", "osaka-japan" and "nagoya-japan"?

##### Try to search for the artist/band `"Scorpions"`.

###### Does it present as result "Scorpions"?

##### Try to search for the member `"Jimi Hendrix"`.

###### Does it present as result the artist/band "The Jimi Hendrix Experience"?

##### Try to search for the member `"Phil Collins"`.

###### Does it present as result "Phil Collins" and "Genesis"?

##### Try to search for the location `"london-uk"`.

###### Does it present as result at least "Pink Floyd", "Led Zeppelin", "Aerosmith", "Alec Benjamin", "Nickelback", "Eagles", "Linkin Park" and "Coldplay"?

##### Try to search for the artist/band `"queen"`.

###### Does it handle the case-insensitive and presents as result "Queen" and "Queensland"?

##### Try to search for the first album date `"05-08-1967"`.

###### Does it present as result "Pink Floyd"?

##### Try to search for the creation date `"1973"`.

###### Does it present as result "ACDC"?

##### Try to search for the creation date `"1965"`.

###### Does it present as result "Scorpions" and "Pink Floyd"?

##### Start typing an artist/band beginning with `"G"`.

###### Is the suggestion helping you find the band you are looking for?

##### Start typing a location of one of the concerts.

###### Is the suggestion helping you find the location you are looking for?

##### Try to search for an artist/band member beginning with `"R"`.

###### Is the suggestion helping you find the artist/band you are looking for?

##### Try to search for a creation date of an artist/band.

###### Is the suggestion helping you find the artist/band you are looking for?

###### As an auditor, is this project up to every standard? If not, why are you failing the project?(Empty Work, Incomplete Work, Invalid compilation, Cheating, Crashing, Leaks)

#### Basic

###### +Does the code obey the [good practices](../../../good-practices/README.md)?

###### +Are the instructions in the website clear?

###### +Does the project run using an API?

#### Social

###### +Did you learn anything from this project?

###### +Can it be open-sourced / be used for other sources?

###### +Would you recommend/nominate this program as an example for the rest of the school?

---