; The demo also exercises Lisp comments.
(begin
  (define square (lambda (x) (* x x)))
  (define-function factorial (n)
    (if (= n 0)
        1
        (* n (factorial (- n 1)))))
  (print (square 7))
  (print (factorial 5))
  (print (car (cons 99 '(1 2 3))))
  (print (if (> 10 3) "yes" "no"))
  (print (length '(snow lisp works)))
  (square 12))
