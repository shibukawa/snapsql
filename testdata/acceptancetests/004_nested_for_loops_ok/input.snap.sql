/*#
function_name: insert_all_sub_departments
parameters:
  departments:
  - department_name: string
    department_code: int
    sub_departments:
    - id: int
      name: string
*/
INSERT INTO sub_departments (id, name, department_code, department_name)
VALUES
/*# for dept : departments */
    /*# for sub : dept.sub_departments */
    (/*= dept.department_code + "-" + sub.id */'1-101', /*= sub.name */'Engineering Team A', /*= dept.department_code */1, /*= dept.department_name */'Engineering')
    /*# end */
/*# end */;
