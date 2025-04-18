SET search_path = hospital;

SELECT 
    specialization, 
    AVG(EXTRACT(YEAR FROM AGE(birth_date))) AS avg_age
FROM 
    Doctors
GROUP BY 
    specialization
ORDER BY 
    specialization;