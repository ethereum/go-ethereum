## Welcome to the go-ethereum website!

This is the repository for the `go-ethereum` website. All the website code is held here in the `website` branch. If you are looking for `go-ethereum` source code you need to switch to the `master` branch.

The purpose of the go-ethereum website is to provide the necessary documentation and supporting information to help users to get up to speed with using go-ethereum (aka "Geth"). The website is maintained by a team of developers but community contributions are also very welcome.

## Contributing

Contributions from the community are very welcome. Please contribute by cloning the `go-ethereum` repository, checking out the `website` branch and raising pull requests to be reviewed and merged by the repository maintainers. Issues can be raised in the main `go-ethereum` repository using the prefix `[website]: ` in the title.

### The geth.ethereum.org stack

geth.ethereum.org is a [Next.js](https://nextjs.org/) project bootstrapped with [`create-next-app`](https://github.com/vercel/next.js/tree/canary/packages/create-next-app). The following tools were used to build the site:

- [Node.js](https://nodejs.org/)
- [React](https://reactjs.org/) - A JavaScript library for building component-based user interfaces
- [Typescript](https://www.typescriptlang.org/) - TypeScript is a strongly typed programming language that builds on JavaScript
- [Chakra UI](https://chakra-ui.com/) - A UI library (Migration in progress)
- [Algolia](https://www.algolia.com/) - Site indexing, rapid intra-site search results, and search analytics. [Learn more on how we implement Algolia for site search](./docs/ALGOLIA_DOCSEARCH.md).
  - Primary implementation: `/src/components/Search/index.ts`
- [Netlify](https://www.netlify.com/) - DNS management and primary host for `master` build.

#### Learn more

To learn more about the stack, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.
- Recommended [free tutorial to learn ChakraUI](https://egghead.io/courses/build-a-modern-user-interface-with-chakra-ui-fac68106).

### Repository structure

The website code is organized with a top-level `docs` folder that contains all the documentation pages as markdown files. Inside `docs` are subdirectories used to divide the docs by theme (e.g. `getting-started`, `fundamentals`, `developers` etc). Website code is in `src`, and assets including images are in `public`.

### Adding a new documentation page

Documentation pages are located in the `/docs` folder in the root directory of the project. The docs pages are all markdown files. When you want to add a new page, add the new file in the appropriate folder in the `/docs` page. `index.md` files will be the default page for a directory, and `{pagename}.md` will define subpages for a directory.

After adding a page, you will also need to list it in `/src/data/documentation-links.yaml`. **This file defines the documentation structure which you will see on the left sidebar in the documentation pages**. Take into account that if you update the `/docs` structure or remove a doc, you should also update this file to avoid navigation issues.

#### Adding notes to a doc

Notes in documentation pages are highlighted boxes (color depend on the current set dark/light theme). To add a note, wrap the note text in `<Note>` tage as follows:

```markdown
<Note>text to include in note</Note>
```

<img width="809" alt="Screen Shot 2023-01-04 at 18 22 06" src="https://user-images.githubusercontent.com/948922/210652463-1fc0370e-815c-427d-9eff-64199a300460.png">

> Example Note from [Account Management with Clef](https://geth.ethereum.org/docs/fundamentals/account-management) doc.

#### Images

Images should be saved to `public/images/docs` and included in the markdown as follows:

```markdown
![alt-text](/images/docs/image-title.png)
```

#### Frontmatter metadata

`title` and `description` are **required** metadata props for a post: `title` will generate the main heading on the doc page and `description` is used for SEO purposes, to serve as a concise and appropriate description of the content.

```
---
title: Go API
description: Introduction to the Go packages that allow Geth to be used in Go native applications.
---
```

> Example of the metadata for a sample post.

### Building locally

To check a new page it is helpful to build the site locally and see how it behaves in the browser. First, run the development server:

```bash
npm run dev
# or
yarn dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `pages/index.tsx`. The page auto-updates as you edit the file.

### Review and merge

PRs will be reviewed by the website maintainers and merged if they improve the website. For substantial changes it is best to reach out to the team by raising a GH issue for discussion first.
