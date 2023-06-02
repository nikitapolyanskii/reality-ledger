# Reality-based UTXO ledger
A repository for building a reality-based ledger using the UTXO (Unspent Transaction Output) model.



---
## Draw a randomly constructed ledger
Use `test_draw.go` for constructing a random ledger and creating  (`.gv`) file. Using Graphviz's `dot` Command-Line Tool one can draw the obtained ledger in (`.svg`) file.

### Prerequisite: Graphviz's `dot` Command-Line Tool

To visualize and convert Graphviz DOT files (`.gv`) to various formats, including SVG, you'll need to have Graphviz installed on your system. Graphviz is an open-source graph visualization software package. One of its command-line tools, `dot`, is used to render graphs and generate output in different formats.

#### Installation Instructions

To install Graphviz, follow these steps:

1. Visit the official Graphviz website: [https://www.graphviz.org/](https://www.graphviz.org/)
2. Download and install the appropriate version of Graphviz for your operating system.
3. Make sure to add the Graphviz binaries to your system's PATH environment variable, so you can access the `dot` command from anywhere in the command-line interface.

### Using `dot` to Convert DOT Files to SVG

Once you have Graphviz installed, you can use the `dot` command-line tool to convert Graphviz DOT files (`.gv`) to SVG format.

To convert a DOT file to SVG, open a terminal or command prompt and navigate to the directory containing the DOT file. Then run the following command:

```
dot -Tsvg -O input.gv
```

This command will generate an SVG file with the same name as the input DOT file, but with the `.svg` extension.

You can then open the SVG file in a web browser or any SVG-compatible viewer to visualize the graph.

---
