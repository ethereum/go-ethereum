import fs from "fs";
import { assertHardhatInvariant } from "./errors";
import { flagParallelChildren, TaskProfile } from "./task-profiling";

export interface Flamegraph {
  name: string;
  value: number;
  children: Flamegraph[];
  parallel: boolean;
}

export function profileToFlamegraph(profile: TaskProfile): Flamegraph {
  assertHardhatInvariant(
    profile.end !== undefined,
    `Formatting invalid task profile for ${profile.name}. No end was recorded.`
  );

  return {
    name: profile.name,
    // We assume this is a safe int, which is ok unless a task runs for months
    value: Number(profile.end - profile.start),
    children: profile.children.map((c) => profileToFlamegraph(c)),
    parallel: profile.parallel === true,
  };
}

/**
 * Merges compatible children of toFold and its children.
 *
 * Compatible in a traditional Flamegraph means having the same name. We
 * modified that notion and also require their `parallel` flag to have the same
 * value. This means that parallel and non-parallel calls to the same function
 * are shown with two Flamegraph "blocks".
 *
 * The parallel block shows the max running time, instead of the sum of them.
 **/
function foldFramegraph(toFold: Flamegraph): Flamegraph {
  if (toFold.children.length === 0) {
    return {
      name: toFold.name,
      value: toFold.value,
      children: [],
      parallel: toFold.parallel,
    };
  }

  const children = toFold.children.map((c) => foldFramegraph(c));
  children.sort((a, b) =>
    a.parallel === b.parallel ? 0 : b.parallel ? -1 : 1
  );
  children.sort((a, b) => a.name.localeCompare(b.name));

  const foldedChildren = [children[0]];
  const mergedChildren = new Set<number>();
  for (let i = 1; i < children.length; i++) {
    const latest = foldedChildren[foldedChildren.length - 1];
    if (
      children[i].name === latest.name &&
      children[i].parallel === latest.parallel
    ) {
      if (latest.parallel) {
        latest.value = Math.max(latest.value, children[i].value);
      } else {
        latest.value += children[i].value;
      }

      latest.children.push(...children[i].children);
      mergedChildren.add(foldedChildren.length - 1);
    } else {
      foldedChildren.push(children[i]);
    }
  }

  for (const i of mergedChildren.values()) {
    foldedChildren[i] = foldFramegraph(foldedChildren[i]);
  }

  return {
    name: toFold.name,
    value: toFold.value,
    children: foldedChildren,
    parallel: toFold.parallel,
  };
}

export function createFlamegraphHtmlFile(flamegraph: Flamegraph): string {
  const content = getFlamegraphFileContent(foldFramegraph(flamegraph));
  const path = "flamegraph.html";

  fs.writeFileSync(path, content, { encoding: "utf8" });

  return path;
}

function getFlamegraphFileContent(flamegraph: Flamegraph): string {
  const data = JSON.stringify(foldFramegraph(flamegraph), undefined, 2);
  return `<!-- Based on d3-flamegraph's example. See its license in: https://github.com/spiermar/d3-flame-graph/blob/master/LICENSE -->
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">

    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css">
    <link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/gh/spiermar/d3-flame-graph@2.0.3/dist/d3-flamegraph.css">

    <style>
    /* Space out content a bit */
    body {
      padding-top: 20px;
      padding-bottom: 20px;
    }

    /* Custom page header */
    .header {
      padding-bottom: 20px;
      padding-right: 15px;
      padding-left: 15px;
      border-bottom: 1px solid #e5e5e5;
    }

    /* Make the masthead heading the same height as the navigation */
    .header h3 {
      margin-top: 0;
      margin-bottom: 0;
      line-height: 40px;
    }

    /* Customize container */
    .container {
      max-width: 990px;
    }
    </style>

    <title>Hardhat task flamegraph</title>
  </head>
  <body>
    <div class="container">
      <div class="header clearfix">
        <nav>
          <div class="pull-right">
            <form class="form-inline" id="form">
              <a class="btn" href="javascript: resetZoom();">Reset zoom</a>
              <a class="btn" href="javascript: clear();">Clear</a>
              <div class="form-group">
                <input type="text" class="form-control" id="term">
              </div>
              <a class="btn btn-primary" href="javascript: search();">Search</a>
            </form>
          </div>
        </nav>
        <h3 class="text-muted">Hardhat task flamegraph</h3>
      </div>
      <div id="chart">
      </div>
      <hr>
      <div id="details">
      </div>
    </div>

    <!-- D3.js -->
    <script src="https://d3js.org/d3.v4.min.js" charset="utf-8"></script>

    <!-- d3-tip -->
    <script type="text/javascript" src="https://cdnjs.cloudflare.com/ajax/libs/d3-tip/0.9.1/d3-tip.min.js"></script>

    <!-- d3-flamegraph -->
    <script type="text/javascript" src="https://cdn.jsdelivr.net/gh/spiermar/d3-flame-graph@2.0.3/dist/d3-flamegraph.min.js"></script>

    <script type="text/javascript">
    const flameGraph = d3.flamegraph()
      .width(960)
      .cellHeight(18)
      .transitionDuration(750)
      .minFrameSize(5)
      .transitionEase(d3.easeCubic)
      .sort(true)
      .title("")
      .onClick(onClick)
      .differential(false)
      .selfValue(false);

    function label(d) {
      if (d.data.parallel) {
        return "(multiple parallel runs) task: " + d.data.name + ", max time: " + readableTime(d.data.value);
      }

      return "task: " + d.data.name + ", time: " + readableTime(d.data.value);
    }

    function readableTime(t) {
      const NANOSECONDS_TO_MILLISECONDS = 1_000_000;
      const NANOSECONDS_TO_SECONDS = 1_000_000_000;

      if (t < NANOSECONDS_TO_MILLISECONDS) {
        return t + "ns";
      }

      if (t < NANOSECONDS_TO_SECONDS) {
          return (t / NANOSECONDS_TO_MILLISECONDS).toFixed(4) + "ms";
      }

      return (t / NANOSECONDS_TO_SECONDS).toFixed(4) + "s";
    }

    const tip = d3.tip()
      .direction("s")
      .offset([8, 0])
      .attr('class', 'd3-flame-graph-tip')
      .html(label);

    flameGraph.tooltip(tip);

    const details = document.getElementById("details");
    flameGraph.setDetailsElement(details);

    flameGraph.label(label);

    flameGraph.setColorMapper(function(d, originalColor) {
      if (d.highlight) {
        return '#E600E6';
      }

      if (d.data.parallel) {
        return '#1478eb'
      }

      return "#EB5414"
    });

    d3.select("#chart")
          .datum(${data})
          .call(flameGraph);

    document.getElementById("form").addEventListener("submit", function(event){
      event.preventDefault();
      search();
    });

    function search() {
      const term = document.getElementById("term").value;
      flameGraph.search(term);
    }

    function clear() {
      document.getElementById('term').value = '';
      flameGraph.clear();
    }

    function resetZoom() {
      flameGraph.resetZoom();
    }

    function onClick(d) {
      console.info("Clicked on " + d.data.name);
    }
    </script>
  </body>
</html>
`;
}

/**
 * Converts the TaskProfile into a flamegraph, saves it, and returns its path.
 */
export function saveFlamegraph(profile: TaskProfile): string {
  flagParallelChildren(profile);
  const flamegraph = profileToFlamegraph(profile);
  return createFlamegraphHtmlFile(flamegraph);
}
