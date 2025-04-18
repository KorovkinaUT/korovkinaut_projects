SET search_path = hospital;

SELECT 
    surname || ' ' || name AS full_name, 
    EXTRACT(YEAR FROM AGE(birth_date)) AS age
FROM 
    Doctors
ORDER BY 
    age
LIMIT 5;