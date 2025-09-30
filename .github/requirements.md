## groupie-tracker

### Objectives

Groupie Trackers consists of receiving a given API and manipulating the data contained in it in order to create a website displaying the information.

- You will be given an [API](https://groupietrackers.herokuapp.com/api), that is made up of four parts:

  1. `artists`: Contains information about some bands and artists like their name(s), image, in which year they began their activity, the date of their first album and the members.

  2. `locations`: Contains their last and/or upcoming concert locations.

  3. `dates`: Contains their last and/or upcoming concert dates.

  4. `relation`: Links the data of all the other parts, `artists`, `dates` and `locations`.

- Given all this, you should build a user-friendly website where you can display the band's info through several data visualizations (examples : blocks, cards, tables, lists, pages, graphics, etc). It is up to you to decide how you will display it.

- This project also focuses on the creation and visualization of events/actions.

  - The event/action you need to implement is a client call to the server (client-server). We can say that it is a feature of your choice that needs to trigger an action. This action necessitates communication with the server in order to receive information ([request-response](https://en.wikipedia.org/wiki/Request%E2%80%93response)).
  - An event consists of a system that responds to some kind of action triggered by the client, time, or any other factor.

### Instructions

- The backend must be written in **Go**.
- The website and server cannot crash at any time.
- All the pages must work correctly, and you must take care of any errors.
- The code must respect the [**good practices**](../good-practices/README.md).
- It is recommended to have **test files** for [unit testing](https://go.dev/doc/tutorial/add-a-test).

### Allowed packages

- Only the [standard Go](https://golang.org/pkg/) packages are allowed.

### Usage

- You can see an example of a RESTful API [here](https://rickandmortyapi.com/)

This project will help you learn about:

- Manipulation and storage of data.
- [JSON](https://www.json.org/json-en.html) files and format.
- HTML.
- Event creation and visualization.
- [Client-server](https://developer.mozilla.org/en-US/docs/Learn/Server-side/First_steps/Client-Server_overview).

---

## groupie-tracker-filters

### Objectives

You must follow the same [principles](../README.md) as the first subject.

- Groupie Tracker Filters consists on letting the user filter the artists/bands that will be shown.

- Your project must incorporate at least these four filters:

  - filter by creation date
  - filter by first album date
  - filter by number of members
  - filter by locations of concerts

- Your filters must be of at least these two types:
  - a range filter (filters the results between two values)
  - a check box filter (filters the results by one or multiple selection)

### Example

Here is an example of both types of filters:

![image](filters_example.png).

### Hints

- You have to pay attention to the locations. For example Seattle, Washington, USA **is part of** Washington, USA.

### Instructions

- The backend must be written in **Go**.
- You must handle website errors.
- The code must respect the [good practices](../../good-practices/README.md)
- It is recommended to have **test files** for [unit testing](https://go.dev/doc/tutorial/add-a-test).

### Allowed packages

- Only the [standard Go](https://golang.org/pkg/) packages are allowed.

This project will help you learn about:

- Manipulation, display and storage of data
- Event creation and display
- JSON files and format
- Go routines

---

## groupie-tracker-search-bar

### Objectives

You must follow the same [principles](../README.md) as the first subject.

Groupie tracker search bar consists of creating a functional program that searches, inside your website, for a specific text input.
So the focus of this project is to create a way for the client to search a member or artist or any other attribute in the data system you made.

- The program should handle at least these search cases :
  - artist/band name
  - members
  - locations
  - first album date
  - creation date
- The program must handle search input as case-insensitive.
- The search bar must have typing suggestions as you write.
  - The search bar must identify and display in each suggestion the individual type of the search cases. (ex: Freddie Mercury -> member)
  - For example if you start writing `"phil"` it should appear as suggestions `Phil Collins - member` and `Phil Collins - artist/band`. This is just an example of a display.

### Example

Lets imagine you have created a card system to display the band data. The user can directly search for the band he needs. Here is an example:

- While the user is typing for the member he desires to see, the search bar gives the suggestion of all the possible options.

![image](searchExample.png)

### Instructions

- The program must be written in **Go**.
- The code must respect the [**good practices**](../../good-practices/README.md).

### Allowed packages

- Only the [standard Go](https://golang.org/pkg/) packages are allowed

This project will help you learn about :

- Manipulation, display and storage of data.
- HTML.
- [Events](https://developer.mozilla.org/en-US/docs/Learn/JavaScript/Building_blocks/) creation and display.
- JSON files and format.

---

## groupie-tracker-search-bar

### Objectives

You must follow the same [principles](../README.md) as the first subject.

Groupie tracker search bar consists of creating a functional program that searches, inside your website, for a specific text input.
So the focus of this project is to create a way for the client to search a member or artist or any other attribute in the data system you made.

- The program should handle at least these search cases :
  - artist/band name
  - members
  - locations
  - first album date
  - creation date
- The program must handle search input as case-insensitive.
- The search bar must have typing suggestions as you write.
  - The search bar must identify and display in each suggestion the individual type of the search cases. (ex: Freddie Mercury -> member)
  - For example if you start writing `"phil"` it should appear as suggestions `Phil Collins - member` and `Phil Collins - artist/band`. This is just an example of a display.

### Example

Lets imagine you have created a card system to display the band data. The user can directly search for the band he needs. Here is an example:

- While the user is typing for the member he desires to see, the search bar gives the suggestion of all the possible options.

![image](searchExample.png)

### Instructions

- The program must be written in **Go**.
- The code must respect the [**good practices**](../../good-practices/README.md).

### Allowed packages

- Only the [standard Go](https://golang.org/pkg/) packages are allowed

This project will help you learn about :

- Manipulation, display and storage of data.
- HTML.
- [Events](https://developer.mozilla.org/en-US/docs/Learn/JavaScript/Building_blocks/) creation and display.
- JSON files and format.