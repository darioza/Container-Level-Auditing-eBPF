MAINFILE = paper
LATEX = pdflatex
LATEXFLAGS = -halt-on-error
BIBTEX = bibtex
TRASHOBJECTS = *.aux *.blg *.bbl *.log *.out
DISTTRASHOBJECTS = *.pdf
OUTPUTDIR = _build

all: clean paper

.PHONY: clean
clean:
	rm -f $(TRASHOBJECTS)

.PHONY: distclean
distclean: clean
	rm -f $(DISTTRASHOBJECTS) &> /dev/null
	rm -f $(OUTPUTDIR)/$(DISTTRASHOBJECTS) &> /dev/null

.PHONY: paper
paper:
	$(LATEX) -shell-escape $(LATEXFLAGS) $(MAINFILE).tex
	$(BIBTEX) $(MAINFILE).aux
	$(LATEX) -shell-escape $(LATEXFLAGS) $(MAINFILE).tex
	$(LATEX) -shell-escape $(LATEXFLAGS) $(MAINFILE).tex
	mv $(MAINFILE).pdf $(OUTPUTDIR)/
