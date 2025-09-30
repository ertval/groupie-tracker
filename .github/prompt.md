Lets follow **test driven development** principles to implement the project as required in the `requirements.md`. Continue where you left the last time. Check `todo.md` for the next steps. Dont stop untill you finish all the steps.
- Create gitignore and license files and documentation, Have a readme file that explains the project structure, setup instructions, and any other relevant information that you keep updated after each important change.
- Write unit tests for all new functionality.
- Do it in small, manageable steps, writing tests for each piece of functionality before implementing it, commit to git after each step.
- Organize the project in a modular way, separating different components and functionalities into their own files and directories.
- Keep a `todo.md` file with the steps of the implementation. **Update it regularly** so you know what is the current state. Also include any new tasks that arise during development and keep detailed documentation of the implementation process.
- In the end write integration comprehensive tests based on `audit.md` to ensure all aspects of the requirements are covered, including edge cases and test all described inputs of the `audit.md` file, only then the project can be considered complete.

---

 Write end to end tests and check that everything works as intended, use #file:audit.md  and #file:requirements.md  for functionality reference. 
 - Check that all templates load correctly
 - Check all visually and functionally using mcp:playwright
 - Save tests in tests folder
 - Update documentation with current state

 ---

 ## We need to implement the following changes:
 - In the location template in the end there is list of most popular locations, it is not working correctly, it should show the locations with the most concerts in total. Fix this.
 - The storage and service layers are a bit messy and complicated, we need to refactor them to be more clear and simple. Use only one package, have a single store struct that handles all the data, and a single service struct that handles all the business logic. Remove any unnecessary abstractions or layers. Make sure the code is easy to read and understand.
 - Write comprehensive tests for the refactored code, covering all the main functionalities and edge cases. Make sure the tests are easy to read and understand, and provide good coverage of the codebase. Update existing tests as needed to reflect the changes made during the refactoring process.
 - Update the readme file to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.
 ### Make sure the project is well organized and structured, with clear separation of concerns and responsibilities. Use appropriate naming conventions and file structures to make it easy to navigate and understand the codebase.

# Clean UP and Optimization
- Remove any unused code, comments, or files that are no longer needed.
- Remove older versions of the files that you changed in the last refactoring. Keep only the simplified versions.
- Rename everything to remove the "Simplified" prefix, so that the new files have the proper shorter names.
- Restructure everything again to be more simple and clear, modular, easy to understand, maintainable and testable:
    - storage package: single store struct, all data operations
    - service package: single service struct, all business logic, calculations (all custom computations here: e.g. location stats, totals, etc.)
    - handlers package: single handlers struct, all HTTP handling
    - Update all imports and references accordingly.
-  - Write comprehensive tests for the refactored code, covering all the main functionalities and edge cases. Make sure the tests are easy to read and understand, and provide good coverage of the codebase. Update existing tests as needed to reflect the changes made during the refactoring process.
 - Update the readme file to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.

---

We need to do the following changes:
- If a template is missing a required field, or the templates are not loaded correctly, the server should log an error and return a 500 Internal Server Error response. NOT PRINT SIMPLE HTML PAGE.
- The handler for artist details mux.HandleFunc("/artists/", h.ArtistDetailHandler) should not allow url like /artists/123/extra, it should return 404 for such urls.
- Remove the all extra functionality like search,filter and refresh. Remove all related code, templates, handlers, tests, etc. We want to keep it simple.
- Update the readme file to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.

---

Simplify the data package to have a single store struct and a single service struct (or as few as possible if one is not sufficient). Remove any unnecessary abstractions or layers. Make sure the code is easy to read and understand.
- Move all the API structs to the api package from the data package. Keep them one to one with what the external API returns. No custom fields or modifications. Have also the validation methods here.
- Keep only the repository structs in the data package. These should be simple structs that represent the application's data model. Simplify them as much as possible, removing any unnecessary fields or methods, reduce duplication, and dont recalculate things that can be precomputed once and stored.
- Be consistent with naming conventions, and file structures to make it easy to navigate and understand the codebase.
- Update the coverage html to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.
- Update the readme file to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.

---

Remove the start server function. Create a bakingInfo function that logs all the important information about the server and the data when the server starts creates the clickable link to open the server in the browser. Call this function from main after everything is initialized and before starting the server. Then start the server directly in main. Always use Idiomatic Go patterns with clean architecture in your implementation.

---

- Read **Carefully** all the #codebase to understad what its doing. 
- Then Try to Refactor and Simplify The code following stirct **Idiomatic GO** and KISS principle. 
- If the code is already good dont change it.
- Try to remove and simplify redundant data structures and dublicate code. 
- Dont touch the templates but addapt the #file:repository to them.
- Run all tests after each change and make sure everything is working. 
- Update the readme file to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.

---

│ > Give me a plan hwo would you refactor the data package such that: 1. You only use idiomatic        │
│   GO and KISS Principle. Store your Plan in a md file. 2. Simplify as much as possible the           │
│   package. 3. Have clear seperate data structs for the api calls, that is artist and relation,       │
│   and for my own internal data store. 4. Have a Repository struct that will hold all                 │
│   precomputed data that are need for the templates and handlers to run correctly. These values       │
│   are computed and loaded with LoadData function. 5. Then there are getter methods for this repo     │
│   instance to get the values out and use them directly without any computation at runtime.  

---

Follow **test driven development**, strict **Idiomatic GO** and KISS principles to implement the project as described in the `requirements.md`. 
- Continue with the implementation of the **search-bar** functionality as described in the `requirements.md` file.
- Do not use any javascript, only Go and HTML and CSS.
- Update the templates to integrate the filter functionality, and any other changes needed to make the project work as intended.
- In the end write integration and e2e comprehensive tests based on `audit.md` to ensure all aspects of the requirements are covered, including edge cases and test all described inputs of the `audit.md` file, only then the project can be considered complete.
- Create summary md file with all the changes you made and how you implemented the functionality.
- Update the readme file to reflect the current state of the project, including any changes made during the implementation process.Dont change the overall structure or flow of the document. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.

---

Read the codebase carefully. Check for redundant, dublicate or overly complex code. Check the package and folder/file structure. Then Propose a restructuring/refactoring plan. Think ultrahard and :
- Use only idiomatic go best practices and kiss priciple in your refactoring. 
- Try to find a golden ration between complexity and simplicity (break functionality into smaller parts/packages/files but not overly so). 
- Focus on reducing LOC while maintaining readability, testability and simplicity.
- Store your plan in a md file that starts with your name.

Propose a project restructuring into packages/folders/files such that its simpler and easer to maintain following Idiomatic Go best practices and KISS principles. Think Ultrahard for this and then store your plan in a md file that starts with your name.