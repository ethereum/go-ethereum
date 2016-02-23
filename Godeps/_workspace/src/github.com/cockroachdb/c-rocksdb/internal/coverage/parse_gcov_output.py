import optparse
import re
import sys

from optparse import OptionParser

# the gcov report follows certain pattern. Each file will have two lines
# of report, from which we can extract the file name, total lines and coverage
# percentage.
def parse_gcov_report(gcov_input):
    per_file_coverage = {}
    total_coverage = None

    for line in sys.stdin:
        line = line.strip()

        # --First line of the coverage report (with file name in it)?
        match_obj = re.match("^File '(.*)'$", line)
        if match_obj:
            # fetch the file name from the first line of the report.
            current_file = match_obj.group(1)
            continue

        # -- Second line of the file report (with coverage percentage)
        match_obj = re.match("^Lines executed:(.*)% of (.*)", line)

        if match_obj:
            coverage = float(match_obj.group(1))
            lines = int(match_obj.group(2))

            if current_file is not None:
                per_file_coverage[current_file] = (coverage, lines)
                current_file = None
            else:
                # If current_file is not set, we reach the last line of report,
                # which contains the summarized coverage percentage.
                total_coverage = (coverage, lines)
            continue

        # If the line's pattern doesn't fall into the above categories. We
        # can simply ignore them since they're either empty line or doesn't
        # find executable lines of the given file.
        current_file = None

    return per_file_coverage, total_coverage

def get_option_parser():
    usage = "Parse the gcov output and generate more human-readable code " +\
            "coverage report."
    parser = OptionParser(usage)

    parser.add_option(
        "--interested-files", "-i",
        dest="filenames",
        help="Comma separated files names. if specified, we will display " +
             "the coverage report only for interested source files. " +
             "Otherwise we will display the coverage report for all " +
             "source files."
    )
    return parser

def display_file_coverage(per_file_coverage, total_coverage):
    # To print out auto-adjustable column, we need to know the longest
    # length of file names.
    max_file_name_length = max(
        len(fname) for fname in per_file_coverage.keys()
    )

    # -- Print header
    # size of separator is determined by 3 column sizes:
    # file name, coverage percentage and lines.
    header_template = \
        "%" + str(max_file_name_length) + "s\t%s\t%s"
    separator = "-" * (max_file_name_length + 10 + 20)
    print header_template % ("Filename", "Coverage", "Lines")
    print separator

    # -- Print body
    # template for printing coverage report for each file.
    record_template = "%" + str(max_file_name_length) + "s\t%5.2f%%\t%10d"

    for fname, coverage_info in per_file_coverage.items():
        coverage, lines = coverage_info
        print record_template % (fname, coverage, lines)

    # -- Print footer
    if total_coverage:
        print separator
        print record_template % ("Total", total_coverage[0], total_coverage[1])

def report_coverage():
    parser = get_option_parser()
    (options, args) = parser.parse_args()

    interested_files = set()
    if options.filenames is not None:
        interested_files = set(f.strip() for f in options.filenames.split(','))

    # To make things simple, right now we only read gcov report from the input
    per_file_coverage, total_coverage = parse_gcov_report(sys.stdin)

    # Check if we need to display coverage info for interested files.
    if len(interested_files):
        per_file_coverage = dict(
            (fname, per_file_coverage[fname]) for fname in interested_files
            if fname in per_file_coverage
        )
        # If we only interested in several files, it makes no sense to report
        # the total_coverage
        total_coverage = None

    if not len(per_file_coverage):
        print >> sys.stderr, "Cannot find coverage info for the given files."
        return
    display_file_coverage(per_file_coverage, total_coverage)

if __name__ == "__main__":
    report_coverage()
