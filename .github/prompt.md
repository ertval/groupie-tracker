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
