#!/usr/bin/sbcl --script

(defun summit (lst)
  (if (null lst)
      0
    (let ((x (car lst)))
      (if (null x)
	  (summit (cdr lst))
	(+ x (summit (cdr lst)))))))

(prin1 (summit '(1 2 nil 3)))
(terpri)
