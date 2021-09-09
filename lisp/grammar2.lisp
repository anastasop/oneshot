
(defparameter *simple-grammar*
  '((sentence -> (noun-phrase verb-phrase))
    (noun-phrase -> (Article Noun))
    (verb-phrase -> (Verb noun-phrase))
    (Article -> the a)
    (Noun -> man ball woman table)
    (Verb -> hit took saw liked)))

(defvar *grammar* *simple-grammar*)

(defun rule-lhs (rule) (first rule))

(defun rule-rhs (rule) (rest (rest rule)))

(defun rewrites (category) (rule-rhs (assoc category *grammar*)))

(defun random-elt (choices) (elt choices (random (length choices))))

(defun mappend (fn the-list)
  (apply #'append (mapcar fn the-list)))

(defun generate (phrase)
  (cond ((listp phrase)
	 (mappend #'generate phrase))
	((rewrites phrase)
	 (generate (random-elt (rewrites phrase))))
	(t
	 (list phrase))))

(defun generate-with-if (phrase)
  (if (listp phrase)
      (mappend #'generate phrase)
    (let ((choices (rewrites phrase)))
      (if (null choices)
	  (list phrase)
	(generate (random-elt choices))))))

    
(mapcar (lambda (a) (print (generate-with-if 'sentence))) '(1 2 3 4 5))
