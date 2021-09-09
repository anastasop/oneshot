
(defvar *start* '((fox hen corn) () ->))

(defun src (state) (first state))

(defun dst (state) (second state))

(defun turn (state) (eq (third state) '->))

(defun goal? (state)
  (let ((dst (dst state)))
    (and (find 'fox dst) (find 'hen dst) (find 'corn dst) (not (turn state)) t)))

(defun moves (state)
  (let* ((side (if (turn state) (src state) (dst state))))
    (cond ((= (length side) 3) '(hen))
	   ((equal side '(hen fox)) '(fox hen))
	   ((equal side '(fox hen)) '(fox hen))
	   ((equal side '(hen corn)) '(hen corn))
	   ((equal side '(corn hen)) '(hen corn))
	   (t (append side '(pass))))))

(defun make-move (state move)
  (let ((src (src state))
	(dst (dst state))
	(next (if (turn state) '<- '->)))
	(cond ((find move src) (list (remove move src) (append dst (list move)) next))
	      ((find move dst) (list (append src (list move)) (remove move dst) next))
	      (t (list src dst next)))))

(defun sig (state)
  (let ((num 0)
	(src (src state))
	(dst (dst state)))
    (when (turn state) (incf num 1))
    (when (find 'fox src) (incf num 2))
    (when (find 'hen src) (incf num 4))
    (when (find 'corn src) (incf num 8))
    (when (find 'fox dst) (incf num 16))
    (when (find 'hen dst) (incf num 32))
    (when (find 'corn dst) (incf num 64))
    num))

(defun print-solution (solution)
  (format t "solution:~&~{  ~s~&~}" solution))

(defun solve ()
  (do ((frontier (list (list *start*)))
       (explored (make-hash-table)))
      ((null frontier))
      (let* ((path (pop frontier))
	     (state (first (last path))))
	(if (goal? state)
	    (print-solution path)
	  (progn
	    (setf (gethash (sig state) explored) 1)
            (dolist (move (moves state))
	      (let ((nextstate (make-move state move)))
		(unless (gethash (sig nextstate) explored)
		  (push (append path (list nextstate)) frontier)))))))))

(solve)
