import fs from 'fs';
import matter from 'gray-matter';
import yaml from 'js-yaml';
import { Box, Grid, Stack, Heading, Text } from '@chakra-ui/react';
import ChakraUIRenderer from 'chakra-ui-markdown-renderer';
import ReactMarkdown from 'react-markdown';
import { useRouter } from 'next/router';
import { useEffect } from 'react';
import gfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { ParsedUrlQuery } from 'querystring';
import type { GetStaticPaths, GetStaticProps, NextPage } from 'next';

import MDComponents, { Breadcrumbs, DocsNav, DocumentNav } from '../components/UI/docs';
import { PageMetadata } from '../components/UI';

import { getFileList } from '../utils/getFileList';
import { getLastModifiedDate, getParsedDate } from '../utils';

import { NavLink } from '../types';

import { textStyles } from '../theme/foundations';

const MATTER_OPTIONS = {
  engines: {
    yaml: (s: any) => yaml.load(s, { schema: yaml.JSON_SCHEMA }) as object
  }
};

// This method crawls for all valid docs paths
export const getStaticPaths: GetStaticPaths = () => {
  const paths: string[] = getFileList('docs'); // This is folder that get crawled for valid docs paths. Change if this path changes.

  return {
    paths,
    fallback: false
  };
};

// Reads file data for markdown pages
export const getStaticProps: GetStaticProps = async context => {
  const navLinks = yaml.load(fs.readFileSync('src/data/documentation-links.yaml', 'utf8'));

  const { slug } = context.params as ParsedUrlQuery;
  const filePath = (slug as string[])!.join('/');
  let file;

  // read file
  try {
    file = fs.readFileSync(`${filePath}/index.md`, 'utf-8');
  } catch (error) {
    file = fs.readFileSync(`${filePath}.md`, 'utf-8');
  }

  const lastModified = await getLastModifiedDate(filePath);
  const { data: frontmatter, content } = matter(file, MATTER_OPTIONS);

  return {
    props: {
      frontmatter,
      content,
      navLinks,
      lastModified: getParsedDate(lastModified, {
        month: 'long',
        day: 'numeric',
        year: 'numeric'
      })
    }
  };
};

interface Props {
  frontmatter: {
    [key: string]: string;
  };
  content: string;
  navLinks: NavLink[];
  lastModified: string;
}

const DocPage: NextPage<Props> = ({ frontmatter, content, navLinks, lastModified }) => {
  const router = useRouter();

  useEffect(() => {
    const id = router.asPath.split('#')[1];
    const element = document.getElementById(id);

    if (!element) {
      return;
    }

    element.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }, [router.asPath]);

  return (
    <>
      <PageMetadata title={frontmatter.title} description={frontmatter.description} />

      <main>
        <Grid
          gap={{ base: 4, lg: 8 }}
          templateColumns={{ base: 'repeat(1, 1fr)', lg: '288px 1fr' }}
        >
          <Stack>
            <DocsNav navLinks={navLinks} />
          </Stack>

          <Stack pb={4} width='100%' id='main-content'>
            <Stack mb={16}>
              <Breadcrumbs />
              <Heading as='h1' mt='4 !important' mb={0} {...textStyles.h1}>
                {frontmatter.title}
              </Heading>
              <Text as='span' mt='0 !important'>
                Last edited on {lastModified}
              </Text>
            </Stack>

            <Grid
              gap={{ base: 4, lg: 8 }}
              templateColumns={{ base: 'repeat(1, 1fr)', xl: '1fr 192px' }}
            >
              <Box
                w='min(100%, 768px)'
                sx={{ '*:first-of-type': { marginTop: '0 !important' } }}
                overflow='auto'
              >
                <ReactMarkdown
                  remarkPlugins={[gfm]}
                  rehypePlugins={[rehypeRaw]}
                  components={ChakraUIRenderer(MDComponents)}
                >
                  {content}
                </ReactMarkdown>
              </Box>

              <Stack display={{ base: 'none', xl: 'block' }}>
                <DocumentNav content={content} />
              </Stack>
            </Grid>
          </Stack>
        </Grid>
      </main>
    </>
  );
};

export default DocPage;
