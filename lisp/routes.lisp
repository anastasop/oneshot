#!/usr/bin/sbcl --script

(defparameter routes `((A B C)
                       (B A C D)
                       (C A B D E)
                       (D B C E)
                       (E)))

(defparameter start 'A)
(defparameter finish 'E)

(defun terminal (path) (eq finish (first (last path))))

(do ((frontier (list (list start))))
    ((null frontier))
    (let* ((path (pop frontier))
           (choices (rest (assoc (first (last path)) routes)))
           (unvisited (set-difference choices path)))
      (if (terminal path)
          (progn (prin1 path) (terpri))
        (dolist (visit unvisited)
          (push (append path (list visit)) frontier)))))
