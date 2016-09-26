/*
 A Naive Bayesian Classifier
 Jake Brukhman <jbrukh@gmail.com>

 BAYESIAN CLASSIFICATION REFRESHER: suppose you have a set
 of classes (e.g. categories) C := {C_1, ..., C_n}, and a
 document D consisting of words D := {W_1, ..., W_k}.
 We wish to ascertain the probability that the document
 belongs to some class C_j given some set of training data
 associating documents and classes.

 By Bayes' Theorem, we have that

    P(C_j|D) = P(D|C_j)*P(C_j)/P(D).

 The LHS is the probability that the document belongs to class
 C_j given the document itself (by which is meant, in practice,
 the word frequencies occurring in this document), and our program
 will calculate this probability for each j and spit out the
 most likely class for this document.

 P(C_j) is referred to as the "prior" probability, or the
 probability that a document belongs to C_j in general, without
 seeing the document first. P(D|C_j) is the probability of seeing
 such a document, given that it belongs to C_j. Here, by assuming
 that words appear independently in documents (this being the
 "naive" assumption), we can estimate

    P(D|C_j) ~= P(W_1|C_j)*...*P(W_k|C_j)

 where P(W_i|C_j) is the probability of seeing the given word
 in a document of the given class. Finally, P(D) can be seen as
 merely a scaling factor and is not strictly relevant to
 classificiation, unless you want to normalize the resulting
 scores and actually see probabilities. In this case, note that

    P(D) = SUM_j(P(D|C_j)*P(C_j))

 One practical issue with performing these calculations is the
 possibility of float64 underflow when calculating P(D|C_j), as
 individual word probabilities can be arbitrarily small, and
 a document can have an arbitrarily large number of them. A
 typical method for dealing with this case is to transform the
 probability to the log domain and perform additions instead
 of multiplications:

   log P(C_j|D) ~ log(P(C_j)) + SUM_i(log P(W_i|C_j))

 where i = 1, ..., k. Note that by doing this, we are discarding
 the scaling factor P(D) and our scores are no longer
 probabilities; however, the monotonic relationship of the
 scores is preserved by the log function.
*/
package bayesian

import (
	"encoding/gob"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
)

// defaultProb is the tiny non-zero probability that a word
// we have not seen before appears in the class.
const defaultProb = 0.00000000001

// ErrUnderflow is returned when an underflow is detected.
var ErrUnderflow = errors.New("possible underflow detected")

// Class defines a class that the classifier will filter:
// C = {C_1, ..., C_n}. You should define your classes as a
// set of constants, for example as follows:
//
//    const (
//        Good Class = "Good"
//        Bad Class = "Bad
//    )
//
// Class values should be unique.
type Class string

// Classifier implements the Naive Bayesian Classifier.
type Classifier struct {
	Classes []Class
	learned int // docs learned
	seen    int // docs seen
	datas   map[Class]*classData
}

// serializableClassifier represents a container for
// Classifier objects whose fields are modifiable by
// reflection and are therefore writeable by gob.
type serializableClassifier struct {
	Classes []Class
	Learned int
	Seen    int
	Datas   map[Class]*classData
}

// classData holds the frequency data for words in a
// particular class. In the future, we may replace this
// structure with a trie-like structure for more
// efficient storage.
type classData struct {
	Freqs map[string]int
	Total int
}

// newClassData creates a new empty classData node.
func newClassData() *classData {
	return &classData{
		Freqs: make(map[string]int),
	}
}

// getWordProb returns P(W|C_j) -- the probability of seeing
// a particular word W in a document of this class.
func (d *classData) getWordProb(word string) float64 {
	value, ok := d.Freqs[word]
	if !ok {
		return defaultProb
	}
	return float64(value) / float64(d.Total)
}

// getWordsProb returns P(D|C_j) -- the probability of seeing
// this set of words in a document of this class.
//
// Note that words should not be empty, and this method of
// calulation is prone to underflow if there are many words
// and their individual probabilties are small.
func (d *classData) getWordsProb(words []string) (prob float64) {
	prob = 1
	for _, word := range words {
		prob *= d.getWordProb(word)
	}
	return
}

// NewClassifier returns a new classifier. The classes provided
// should be at least 2 in number and unique, or this method will
// panic.
func NewClassifier(classes ...Class) (c *Classifier) {
	n := len(classes)

	// check size
	if n < 2 {
		panic("provide at least two classes")
	}

	// check uniqueness
	check := make(map[Class]bool, n)
	for _, class := range classes {
		check[class] = true
	}
	if len(check) != n {
		panic("classes must be unique")
	}
	// create the classifier
	c = &Classifier{
		Classes: classes,
		datas:   make(map[Class]*classData, n),
	}
	for _, class := range classes {
		c.datas[class] = newClassData()
	}
	return
}

// NewClassifierFromFile loads an existing classifier from
// file. The classifier was previously saved with a call
// to c.WriteToFile(string).
func NewClassifierFromFile(name string) (c *Classifier, err error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return NewClassifierFromReader(file)
}

// NewClassifierFromReader actually does the deserializing of a Gob encoded classifier.
func NewClassifierFromReader(r io.Reader) (c *Classifier, err error) {
	dec := gob.NewDecoder(r)
	w := new(serializableClassifier)
	err = dec.Decode(w)

	return &Classifier{w.Classes, w.Learned, w.Seen, w.Datas}, err
}

// getPriors returns the prior probabilities for the
// classes provided -- P(C_j).
//
// TODO: There is a way to smooth priors, currently
// not implemented here.
func (c *Classifier) getPriors() (priors []float64) {
	n := len(c.Classes)
	priors = make([]float64, n, n)
	sum := 0
	for index, class := range c.Classes {
		total := c.datas[class].Total
		priors[index] = float64(total)
		sum += total
	}
	if sum != 0 {
		for i := 0; i < n; i++ {
			priors[i] /= float64(sum)
		}
	}
	return
}

// Learned returns the number of documents ever learned
// in the lifetime of this classifier.
func (c *Classifier) Learned() int {
	return c.learned
}

// Seen returns the number of documents ever classified
// in the lifetime of this classifier.
func (c *Classifier) Seen() int {
	return c.seen
}

// WordCount returns the number of words counted for
// each class in the lifetime of the classifier.
func (c *Classifier) WordCount() (result []int) {
	result = make([]int, len(c.Classes))
	for inx, class := range c.Classes {
		data := c.datas[class]
		result[inx] = data.Total
	}
	return
}

// Observe should be used when word-frequencies have been already been learned
// externally (e.g., hadoop)
func (c *Classifier) Observe(word string, count int, which Class) {
	data := c.datas[which]
	data.Freqs[word] += count
	data.Total += count
}

// Learn will accept new training documents for
// supervised learning.
func (c *Classifier) Learn(document []string, which Class) {
	data := c.datas[which]
	for _, word := range document {
		data.Freqs[word]++
		data.Total++
	}
	c.learned++
}

// LogScores produces "log-likelihood"-like scores that can
// be used to classify documents into classes.
//
// The value of the score is proportional to the likelihood,
// as determined by the classifier, that the given document
// belongs to the given class. This is true even when scores
// returned are negative, which they will be (since we are
// taking logs of probabilities).
//
// The index j of the score corresponds to the class given
// by c.Classes[j].
//
// Additionally returned are "inx" and "strict" values. The
// inx corresponds to the maximum score in the array. If more
// than one of the scores holds the maximum values, then
// strict is false.
//
// Unlike c.Probabilities(), this function is not prone to
// floating point underflow and is relatively safe to use.
func (c *Classifier) LogScores(document []string) (scores []float64, inx int, strict bool) {
	n := len(c.Classes)
	scores = make([]float64, n, n)
	priors := c.getPriors()

	// calculate the score for each class
	for index, class := range c.Classes {
		data := c.datas[class]
		// c is the sum of the logarithms
		// as outlined in the refresher
		score := math.Log(priors[index])
		for _, word := range document {
			score += math.Log(data.getWordProb(word))
		}
		scores[index] = score
	}
	inx, strict = findMax(scores)
	c.seen++
	return scores, inx, strict
}

// ProbScores works the same as LogScores, but delivers
// actual probabilities as discussed above. Note that float64
// underflow is possible if the word list contains too
// many words that have probabilities very close to 0.
//
// Notes on underflow: underflow is going to occur when you're
// trying to assess large numbers of words that you have
// never seen before. Depending on the application, this
// may or may not be a concern. Consider using SafeProbScores()
// instead.
func (c *Classifier) ProbScores(doc []string) (scores []float64, inx int, strict bool) {
	n := len(c.Classes)
	scores = make([]float64, n, n)
	priors := c.getPriors()
	sum := float64(0)
	// calculate the score for each class
	for index, class := range c.Classes {
		data := c.datas[class]
		// c is the sum of the logarithms
		// as outlined in the refresher
		score := priors[index]
		for _, word := range doc {
			score *= data.getWordProb(word)
		}
		scores[index] = score
		sum += score
	}
	for i := 0; i < n; i++ {
		scores[i] /= sum
	}
	inx, strict = findMax(scores)
	c.seen++
	return scores, inx, strict
}

// SafeProbScores works the same as ProbScores, but is
// able to detect underflow in those cases where underflow
// results in the reverse classification. If an underflow is detected,
// this method returns an ErrUnderflow, allowing the user to deal with it as
// necessary. Note that underflow, under certain rare circumstances,
// may still result in incorrect probabilities being returned,
// but this method guarantees that all error-less invokations
// are properly classified.
//
// Underflow detection is more costly because it also
// has to make additional log score calculations.
func (c *Classifier) SafeProbScores(doc []string) (scores []float64, inx int, strict bool, err error) {
	n := len(c.Classes)
	scores = make([]float64, n, n)
	logScores := make([]float64, n, n)
	priors := c.getPriors()
	sum := float64(0)
	// calculate the score for each class
	for index, class := range c.Classes {
		data := c.datas[class]
		// c is the sum of the logarithms
		// as outlined in the refresher
		score := priors[index]
		logScore := math.Log(priors[index])
		for _, word := range doc {
			p := data.getWordProb(word)
			score *= p
			logScore += math.Log(p)
		}
		scores[index] = score
		logScores[index] = logScore
		sum += score
	}
	for i := 0; i < n; i++ {
		scores[i] /= sum
	}
	inx, strict = findMax(scores)
	logInx, logStrict := findMax(logScores)

	// detect underflow -- the size
	// relation between scores and logScores
	// must be preserved or something is wrong
	if inx != logInx || strict != logStrict {
		err = ErrUnderflow
	}
	c.seen++
	return scores, inx, strict, err
}

// WordFrequencies returns a matrix of word frequencies that currently
// exist in the classifier for each class state for the given input
// words. In other words, if you obtain the frequencies
//
//    freqs := c.WordFrequencies(/* [j]string */)
//
// then the expression freq[i][j] represents the frequency of the j-th
// word within the i-th class.
func (c *Classifier) WordFrequencies(words []string) (freqMatrix [][]float64) {
	n, l := len(c.Classes), len(words)
	freqMatrix = make([][]float64, n)
	for i, _ := range freqMatrix {
		arr := make([]float64, l)
		data := c.datas[c.Classes[i]]
		for j, _ := range arr {
			arr[j] = data.getWordProb(words[j])
		}
		freqMatrix[i] = arr
	}
	return
}

// WordsByClass returns a map of words and their probability of
// appearing in the given class.
func (c *Classifier) WordsByClass(class Class) (freqMap map[string]float64) {
	freqMap = make(map[string]float64)
	for word, cnt := range c.datas[class].Freqs {
		freqMap[word] = float64(cnt) / float64(c.datas[class].Total)
	}

	return freqMap
}

// Serialize this classifier to a file.
func (c *Classifier) WriteToFile(name string) (err error) {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return c.WriteTo(file)
}

// WriteClassesToFile writes all classes to files.
func (c *Classifier) WriteClassesToFile(rootPath string) (err error) {
	for name, _ := range c.datas {
		c.WriteClassToFile(name, rootPath)
	}
	return
}

func (c *Classifier) WriteClassToFile(name Class, rootPath string) (err error) {
	data := c.datas[name]
	fileName := filepath.Join(rootPath, string(name))
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(file)
	err = enc.Encode(data)
	return
}

// Serialize this classifier to GOB and write to Writer.
func (c *Classifier) WriteTo(w io.Writer) (err error) {
	enc := gob.NewEncoder(w)
	err = enc.Encode(&serializableClassifier{c.Classes, c.learned, c.seen, c.datas})
	return
}

// ReadClassFromFile loads existing class data from a
// file.
func (c *Classifier) ReadClassFromFile(class Class, location string) (err error) {
	fileName := filepath.Join(location, string(class))
	file, err := os.Open(fileName)

	if err != nil {
		return err
	}

	dec := gob.NewDecoder(file)
	w := new(classData)
	err = dec.Decode(w)

	c.learned++
	c.datas[class] = w
	return
}

// findMax finds the maximum of a set of scores; if the
// maximum is strict -- that is, it is the single unique
// maximum from the set -- then strict has return value
// true. Otherwise it is false.
func findMax(scores []float64) (inx int, strict bool) {
	inx = 0
	strict = true
	for i := 1; i < len(scores); i++ {
		if scores[inx] < scores[i] {
			inx = i
			strict = true
		} else if scores[inx] == scores[i] {
			strict = false
		}
	}
	return
}
