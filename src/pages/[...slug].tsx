import fs from 'fs';
import matter from 'gray-matter';
import yaml from 'js-yaml';
import { Stack, Heading } from '@chakra-ui/react';
import ChakraUIRenderer from 'chakra-ui-markdown-renderer';
import ReactMarkdown from 'react-markdown';
import gfm from 'remark-gfm';
import { ParsedUrlQuery } from 'querystring';
import type { GetStaticPaths, GetStaticProps, NextPage } from 'next';

import MDXComponents from '../components/';
import { Breadcrumbs } from '../components/docs'
import { PageMetadata } from '../components/UI';
import { textStyles } from '../theme/foundations';


const MATTER_OPTIONS = {
  engines: {
    yaml: (s: any) => yaml.load(s, { schema: yaml.JSON_SCHEMA }) as object
  }
};

// This method crawls for all valid docs paths
export const getStaticPaths: GetStaticPaths = () => {
  const getFileList = (dirName: string) => {
    let files: string[] = [];
    const items = fs.readdirSync(dirName, { withFileTypes: true });

    for (const item of items) {
      if (item.isDirectory()) {
        files = [...files, ...getFileList(`${dirName}/${item.name}`)];
      } else {
        files.push(`/${dirName}/${item.name}`);
      }
    }

    return files.map(file => file.replace('.md', '')).map(file => file.replace('/index', ''));
  };

  const paths: string[] = getFileList('docs'); // This is folder that get crawled for valid docs paths. Change if this path changes.

  return {
    paths,
    fallback: false
  };
};

// Reads file data for markdown pages
export const getStaticProps: GetStaticProps = async context => {
  const { slug } = context.params as ParsedUrlQuery;
  const filePath = (slug as string[])!.join('/');
  let file;

  try {
    file = fs.readFileSync(`${filePath}.md`, 'utf-8');
  } catch {
    file = fs.readFileSync(`${filePath}/index.md`, 'utf-8');
  }

  const { data: frontmatter, content } = matter(file, MATTER_OPTIONS);

  return {
    props: {
      frontmatter,
      content
    }
  };
};

interface Props {
  frontmatter: {
    [key: string]: string;
  };
  content: string;
}

const DocPage: NextPage<Props> = ({ frontmatter, content }) => {
  return (
    <>
      <PageMetadata title={frontmatter.title} description={frontmatter.description} />

      <main>
        <Stack mb={16}>
          <Breadcrumbs />
          <Heading as='h1' mt='4 !important' mb={0} {...textStyles.header1}>
            {frontmatter.title}
          </Heading>
          {/* <Text as='span' mt='0 !important'>last edited {TODO: get last edited date}</Text> */}
        </Stack>
        <ReactMarkdown remarkPlugins={[gfm]} components={ChakraUIRenderer(MDXComponents)}>{content}</ReactMarkdown>
      </main>
    </>
  );
};

export default DocPage;
