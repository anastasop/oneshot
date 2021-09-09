(defun ask-number()
  (format t "Please enter a number. ")
  (let ((val (read)))
    (if (numberp val)
	val
      (ask-number))))
