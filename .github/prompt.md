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